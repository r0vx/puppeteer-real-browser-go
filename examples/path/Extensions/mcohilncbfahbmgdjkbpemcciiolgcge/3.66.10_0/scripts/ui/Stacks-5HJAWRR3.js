import{o as r,q as o}from"./chunk-SHG7TIBL.js";r();o();var i={getAccount:async({provider:s})=>{let{address:t}=await s?.stacks?.connect();return t},signMessage:async({provider:s,message:t})=>{let{stacks:a}=s||{},{signature:n}=await a?.signMessage({message:t});return n},sendTransaction:async({provider:s,payload:t,cb:a,extraParams:n})=>{try{let{stacks:c}=s||{},{txHash:e}=await c?.signTransaction(t,n);return a&&a(e),e}catch(c){throw c}}},u=i;export{u as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=Stacks-5HJAWRR3.js.map
