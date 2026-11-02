importScripts('wasm_exec.js');
(async () => {
  try {
    const go = new Go();
    const bytes = await (await fetch('ngn.wasm')).arrayBuffer();
    const {instance} = await WebAssembly.instantiate(bytes, go.importObject);
    go.run(instance);
    postMessage({ready:true});
  } catch (error) { postMessage({error:String(error)}); }
})();
onmessage = event => {
  try { postMessage(JSON.parse(ngn(JSON.stringify(event.data)))); }
  catch (error) { postMessage({error:String(error)}); }
};
