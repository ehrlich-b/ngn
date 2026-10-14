package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

func tiltedShape(shape [5]float64, score float64) [5]float64 {
	low, high := -100.0, 100.0
	var probabilities [5]float64
	for iteration := 0; iteration < 100; iteration++ {
		tilt, total, mean := (low+high)/2, 0.0, 0.0
		for k := range shape {
			probabilities[k] = shape[k] * math.Exp(tilt*float64(k)/4)
			total += probabilities[k]
		}
		for k := range shape {
			probabilities[k] /= total
			mean += float64(k) / 4 * probabilities[k]
		}
		if mean < score {
			low = tilt
		} else {
			high = tilt
		}
	}
	return probabilities
}

func runCalibration(trials int, input string, elo0, elo1, alpha, beta float64, minGames, maxGames int, seed int64) {
	fields := strings.Split(input, ",")
	if len(fields) != 5 || trials < 1 || minGames < 2 || maxGames < minGames {
		log.Fatal("invalid calibration options")
	}
	var shape [5]float64
	for k, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil || value <= 0 {
			log.Fatal("calibration shape requires five positive counts")
		}
		shape[k] = value
	}
	lower, upper := math.Log(beta/(1-alpha)), math.Log((1-beta)/alpha)
	rng := rand.New(rand.NewSource(seed))
	rows := []map[string]any{}
	for _, truth := range []float64{elo0, elo1} {
		probabilities := tiltedShape(shape, eloToScore(truth))
		h0, h1, capped, games := 0, 0, 0, 0
		for trial := 0; trial < trials; trial++ {
			var penta [5]int
			decision, n := 0, 0
			for n < maxGames {
				draw, bin := rng.Float64(), 0
				for bin < 4 && draw >= probabilities[bin] {
					draw -= probabilities[bin]
					bin++
				}
				penta[bin]++
				n += 2
				if n >= minGames {
					llr, _, _, _, _ := pentaStats(penta, elo0, elo1)
					if llr >= upper {
						decision = 1
						break
					}
					if llr <= lower {
						decision = -1
						break
					}
				}
			}
			games += n
			switch decision {
			case 1:
				h1++
			case -1:
				h0++
			default:
				capped++
			}
		}
		rows = append(rows, map[string]any{"true_elo": truth, "h0": h0, "h1": h1, "capped": capped,
			"h1_rate": float64(h1) / float64(trials), "mean_games": float64(games) / float64(trials)})
	}
	payload := map[string]any{"trials_per_truth": trials, "seed": seed, "shape": shape, "elo0": elo0,
		"elo1": elo1, "alpha": alpha, "beta": beta, "min_games": minGames, "max_games": maxGames,
		"llr": "cmd/sprt.pentaStats", "results": rows}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}
