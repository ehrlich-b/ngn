use std::{
    env, fs,
    path::{Path, PathBuf},
};

use bullet_lib::{
    game::{
        inputs::{ChessBucketsMirrored, get_num_buckets},
        outputs::MaterialCount,
    },
    nn::{
        InitSettings, Shape,
        optimiser::{AdamW, AdamWParams},
    },
    trainer::{
        save::SavedFormat,
        schedule::{TrainingSchedule, TrainingSteps, lr, wdl},
        settings::LocalSettings,
    },
    value::{ValueTrainerBuilder, loader::DirectSequentialDataLoader},
};

const BULLET_COMMIT: &str = "629ee50000b2afb7b3337595401c830d3b1e0f42";
const MODEL_SEED: u64 = 26_092_001;
const HIDDEN: usize = 768;
const OUTPUT_BUCKETS: usize = 8;
const SCORE_SCALE: i16 = 400;
const QA: i16 = 255;
const QB: i16 = 64;
const BATCH_SIZE: usize = 16_384;
const LOADER_THREADS: usize = 2;
const BATCH_QUEUE_SIZE: usize = 8;
const INITIAL_LR: f32 = 0.001;
const FINAL_LR: f32 = 0.00005;
const DEPLOYED_VALUES: usize = 2_372_360;
const RAW_BYTES: usize = DEPLOYED_VALUES * 4;
const QUANT_TENSOR_BYTES: usize = DEPLOYED_VALUES * 2;
const QUANT_BYTES: usize = QUANT_TENSOR_BYTES + 48;

#[rustfmt::skip]
const BUCKET_LAYOUT: [usize; 32] = [
    1, 1, 0, 0,
    2, 2, 2, 2,
    3, 3, 3, 3,
    3, 3, 3, 3,
    3, 3, 3, 3,
    3, 3, 3, 3,
    3, 3, 3, 3,
    3, 3, 3, 3,
];

const INPUT_BUCKETS: usize = get_num_buckets(&BUCKET_LAYOUT);

#[derive(Clone, Copy)]
struct Mode {
    name: &'static str,
    positions: u64,
    updates: usize,
    checkpoint_updates: usize,
}

fn mode(value: &str) -> Option<Mode> {
    match value {
        "smoke" => Some(Mode {
            name: "smoke",
            positions: 1_000_000,
            updates: 256,
            checkpoint_updates: 256,
        }),
        "pilot" => Some(Mode {
            name: "pilot",
            positions: 1_000_000,
            updates: 1_024,
            checkpoint_updates: 128,
        }),
        "main" => Some(Mode {
            name: "main",
            positions: 20_000_000,
            updates: 16_384,
            checkpoint_updates: 512,
        }),
        _ => None,
    }
}

fn fail(message: impl AsRef<str>) -> ! {
    eprintln!("NGN_K4_TRAIN_FAIL {}", message.as_ref());
    std::process::exit(1);
}

fn f32_values(path: &Path) -> Vec<f32> {
    let bytes =
        fs::read(path).unwrap_or_else(|error| fail(format!("read {}: {error}", path.display())));
    if bytes.len() != RAW_BYTES {
        fail(format!(
            "{} bytes={} expected={RAW_BYTES}",
            path.display(),
            bytes.len()
        ));
    }
    let values: Vec<f32> = bytes
        .chunks_exact(4)
        .map(|word| f32::from_le_bytes(word.try_into().unwrap()))
        .collect();
    if let Some((index, value)) = values
        .iter()
        .enumerate()
        .find(|(_, value)| !value.is_finite())
    {
        fail(format!("{} raw[{index}] nonfinite={value}", path.display()));
    }
    values
}

fn quantise(value: f32, scale: i16) -> i16 {
    let rounded = (f64::from(value) * f64::from(scale)).round();
    if !(f64::from(i16::MIN)..=f64::from(i16::MAX)).contains(&rounded) {
        fail(format!("quantisation range value={value} scale={scale}"));
    }
    rounded as i16
}

fn validate_quantised(raw: &[f32], path: &Path) {
    let bytes =
        fs::read(path).unwrap_or_else(|error| fail(format!("read {}: {error}", path.display())));
    if bytes.len() != QUANT_BYTES {
        fail(format!(
            "{} bytes={} expected={QUANT_BYTES}",
            path.display(),
            bytes.len()
        ));
    }
    let sections = [
        (INPUT_BUCKETS * 768 * HIDDEN, QA),
        (HIDDEN, QA),
        (OUTPUT_BUCKETS * 2 * HIDDEN, QB),
        (OUTPUT_BUCKETS, QA * QB),
    ];
    let mut raw_offset = 0usize;
    let mut byte_offset = 0usize;
    for (count, scale) in sections {
        for &value in &raw[raw_offset..raw_offset + count] {
            let expected = quantise(value, scale).to_le_bytes();
            if bytes[byte_offset..byte_offset + 2] != expected {
                fail(format!(
                    "{} quantised mismatch tensor_index={} got={:?} expected={expected:?}",
                    path.display(),
                    raw_offset,
                    &bytes[byte_offset..byte_offset + 2],
                ));
            }
            raw_offset += 1;
            byte_offset += 2;
        }
    }
    if raw_offset != DEPLOYED_VALUES || byte_offset != QUANT_TENSOR_BYTES {
        fail("internal tensor count mismatch");
    }
    if bytes[QUANT_TENSOR_BYTES..]
        .iter()
        .enumerate()
        .any(|(index, byte)| *byte != b"bullet"[index % 6])
    {
        fail(format!("{} invalid Bullet trailer", path.display()));
    }
}

fn validate_checkpoint(path: &Path) {
    let raw = f32_values(&path.join("raw.bin"));
    validate_quantised(&raw, &path.join("quantised.bin"));
    for state in ["momentum.bin", "velocity.bin"] {
        let state_path = path.join("optimiser_state").join(state);
        let metadata = fs::metadata(&state_path).unwrap_or_else(|error| {
            fail(format!("optimiser state {}: {error}", state_path.display()))
        });
        if metadata.len() == 0 || metadata.len() % 4 != 0 {
            fail(format!(
                "invalid optimiser state {} bytes={}",
                state_path.display(),
                metadata.len()
            ));
        }
    }
}

fn deployed_raw_format() -> Vec<SavedFormat> {
    vec![
        SavedFormat::id("l0w").transform(|store, weights| {
            let factoriser = store.get("l0f").values.f32().repeat(INPUT_BUCKETS);
            weights
                .into_iter()
                .zip(factoriser)
                .map(|(main, shared)| main + shared)
                .collect()
        }),
        SavedFormat::id("l0b"),
        SavedFormat::id("l1w").transpose(),
        SavedFormat::id("l1b"),
    ]
}

fn deployed_quantised_format() -> Vec<SavedFormat> {
    vec![
        SavedFormat::id("l0w")
            .transform(|store, weights| {
                let factoriser = store.get("l0f").values.f32().repeat(INPUT_BUCKETS);
                weights
                    .into_iter()
                    .zip(factoriser)
                    .map(|(main, shared)| main + shared)
                    .collect()
            })
            .round()
            .quantise::<i16>(QA),
        SavedFormat::id("l0b").round().quantise::<i16>(QA),
        SavedFormat::id("l1w")
            .round()
            .quantise::<i16>(QB)
            .transpose(),
        SavedFormat::id("l1b").round().quantise::<i16>(QA * QB),
    ]
}

fn write_deployed_raw<T, I, O>(trainer: &bullet_lib::value::ValueTrainer<T, I, O>, path: &Path)
where
    T: bullet_trainer::optimiser::OptimiserState<bullet_lib::nn::ExecutionContext>,
    I: bullet_lib::game::inputs::SparseInputType,
{
    let weights = trainer
        .optimiser
        .cpu_weights()
        .unwrap_or_else(|error| fail(format!("read CPU weights: {error:?}")));
    let bytes = weights
        .to_quantised_buffer(&deployed_raw_format(), false)
        .unwrap_or_else(|error| fail(format!("encode deployed raw {}: {error}", path.display())));
    fs::write(path, bytes)
        .unwrap_or_else(|error| fail(format!("write deployed raw {}: {error}", path.display())));
}

fn strict_save<T, I, O>(trainer: &bullet_lib::value::ValueTrainer<T, I, O>, root: &Path)
where
    T: bullet_trainer::optimiser::OptimiserState<bullet_lib::nn::ExecutionContext>,
    I: bullet_lib::game::inputs::SparseInputType,
{
    fs::create_dir(root)
        .unwrap_or_else(|error| fail(format!("create {}: {error}", root.display())));
    let optimiser_state = root.join("optimiser_state");
    fs::create_dir(&optimiser_state)
        .unwrap_or_else(|error| fail(format!("create {}: {error}", optimiser_state.display())));
    trainer
        .optimiser
        .write_to_checkpoint(optimiser_state.to_str().unwrap())
        .unwrap_or_else(|error| fail(format!("write optimiser {}: {error:?}", root.display())));
    write_deployed_raw(trainer, &root.join("raw.bin"));
    trainer
        .save_quantised(root.join("quantised.bin").to_str().unwrap())
        .unwrap_or_else(|error| fail(format!("save quantised {}: {error}", root.display())));
    validate_checkpoint(root);
}

fn main() {
    let args: Vec<_> = env::args_os().collect();
    if args.len() != 4 {
        fail(
            "usage: ngn_k4_train smoke|pilot|main TRAIN_BF NEW_OUTPUT_DIRECTORY\n       ngn_k4_train probe CHECKPOINT_DIRECTORY FENS_FILE",
        );
    }
    let mode_text = args[1].to_string_lossy();
    let probe = mode_text == "probe";
    let run_mode = if probe {
        None
    } else {
        Some(mode(&mode_text).unwrap_or_else(|| fail(format!("unknown mode {mode_text}"))))
    };
    if INPUT_BUCKETS != 4 {
        fail(format!("input buckets={INPUT_BUCKETS} expected=4"));
    }

    let mut trainer = ValueTrainerBuilder::default()
        .seed(MODEL_SEED)
        .dual_perspective()
        .optimiser(AdamW)
        .inputs(ChessBucketsMirrored::new(BUCKET_LAYOUT))
        .output_buckets(MaterialCount::<OUTPUT_BUCKETS>)
        .save_format(&deployed_quantised_format())
        .loss_fn(|network_output, target| network_output.sigmoid().squared_error(target))
        .build(|builder, stm_inputs, ntm_inputs, output_buckets| {
            let factoriser =
                builder.new_weights("l0f", Shape::new(HIDDEN, 768), InitSettings::Zeroed);
            let expanded_factoriser = factoriser.repeat(INPUT_BUCKETS);
            let mut l0 = builder.new_affine("l0", 768 * INPUT_BUCKETS, HIDDEN);
            l0.weights = l0.weights + expanded_factoriser;
            let l1 = builder.new_affine("l1", 2 * HIDDEN, OUTPUT_BUCKETS);
            let stm_hidden = l0.forward(stm_inputs).screlu();
            let ntm_hidden = l0.forward(ntm_inputs).screlu();
            l1.forward(stm_hidden.concat(ntm_hidden))
                .select(output_buckets)
        });

    let stricter = AdamWParams {
        max_weight: 0.99,
        min_weight: -0.99,
        ..Default::default()
    };
    trainer.optimiser.set_params_for_weight("l0w", stricter);
    trainer.optimiser.set_params_for_weight("l0f", stricter);

    if probe {
        let checkpoint = PathBuf::from(&args[2]);
        let fen_path = PathBuf::from(&args[3]);
        if !checkpoint.is_dir() {
            fail(format!(
                "probe checkpoint is not a directory: {}",
                checkpoint.display()
            ));
        }
        let fen_text = fs::read_to_string(&fen_path)
            .unwrap_or_else(|error| fail(format!("read FENs {}: {error}", fen_path.display())));
        let fens: Vec<_> = fen_text
            .lines()
            .map(str::trim)
            .filter(|line| !line.is_empty() && !line.starts_with('#'))
            .collect();
        if fens.is_empty() || fens.iter().any(|fen| fen.contains('\t')) {
            fail("probe FEN file must contain nonempty, tab-free positions");
        }
        trainer.load_from_checkpoint(checkpoint.to_str().unwrap());
        for (index, fen) in fens.iter().enumerate() {
            let output = trainer.eval_raw_output(fen);
            if output.len() != 1 || !output[0].is_finite() {
                fail(format!("probe {index} returned {output:?}"));
            }
            println!("NGN_K4_PROBE\t{index}\t{:.9}\t{fen}", output[0]);
        }
        println!(
            "NGN_K4_PROBE_PASS bullet_commit={BULLET_COMMIT} positions={}",
            fens.len()
        );
        return;
    }

    let run_mode = run_mode.unwrap();
    if run_mode.updates % run_mode.checkpoint_updates != 0 {
        fail("updates must divide evenly into checkpoint intervals");
    }
    let train = PathBuf::from(&args[2]);
    let output = PathBuf::from(&args[3]);
    let metadata = fs::metadata(&train)
        .unwrap_or_else(|error| fail(format!("train {}: {error}", train.display())));
    let expected_bytes = run_mode.positions * 32;
    if metadata.len() != expected_bytes {
        fail(format!(
            "train bytes={} expected={expected_bytes}",
            metadata.len()
        ));
    }
    if output.exists() {
        fail(format!("refusing existing output {}", output.display()));
    }
    let candidates = output.join("candidates");
    fs::create_dir_all(&candidates)
        .unwrap_or_else(|error| fail(format!("create candidates: {error}")));

    strict_save(&trainer, &candidates.join("candidate-0"));
    let train_string = train.to_str().unwrap().to_owned();
    let loader = DirectSequentialDataLoader::new(&[&train_string]);
    let total_superbatches = run_mode.updates / run_mode.checkpoint_updates;
    let candidate_output = candidates.to_str().unwrap().to_owned();

    for superbatch in 1..=total_superbatches {
        let schedule = TrainingSchedule {
            net_id: format!("ngn-k4-{}", run_mode.name),
            eval_scale: f32::from(SCORE_SCALE),
            steps: TrainingSteps {
                batch_size: BATCH_SIZE,
                batches_per_superbatch: run_mode.checkpoint_updates,
                start_superbatch: superbatch,
                end_superbatch: superbatch,
            },
            wdl_scheduler: wdl::ConstantWDL { value: 0.0 },
            lr_scheduler: lr::CosineDecayLR {
                initial_lr: INITIAL_LR,
                final_lr: FINAL_LR,
                final_superbatch: total_superbatches,
            },
            save_rate: 1,
        };
        let settings = LocalSettings {
            threads: LOADER_THREADS,
            test_set: None,
            output_directory: &candidate_output,
            batch_queue_size: BATCH_QUEUE_SIZE,
        };
        trainer.run(&schedule, &settings, &loader);
        let automatic = candidates.join(format!("ngn-k4-{}-{superbatch}", run_mode.name));
        write_deployed_raw(&trainer, &automatic.join("raw.bin"));
        validate_checkpoint(&automatic);
        let completed_updates = superbatch * run_mode.checkpoint_updates;
        let candidate = candidates.join(format!("candidate-{completed_updates}"));
        if candidate.exists() {
            fail(format!("candidate already exists {}", candidate.display()));
        }
        fs::rename(&automatic, &candidate).unwrap_or_else(|error| {
            fail(format!(
                "rename {} to {}: {error}",
                automatic.display(),
                candidate.display()
            ))
        });
    }

    println!(
        "NGN_K4_TRAIN_PASS bullet_commit={BULLET_COMMIT} mode={} seed={MODEL_SEED} input_buckets={INPUT_BUCKETS} hidden={HIDDEN} output_buckets={OUTPUT_BUCKETS} result_weight=0 score_weight=1 score_scale={SCORE_SCALE} batch_size={BATCH_SIZE} updates={} presentations={} train_positions={} loader_threads={LOADER_THREADS} queue={BATCH_QUEUE_SIZE} checkpoints={}",
        run_mode.name,
        run_mode.updates,
        BATCH_SIZE * run_mode.updates,
        run_mode.positions,
        total_superbatches + 1,
    );
}
