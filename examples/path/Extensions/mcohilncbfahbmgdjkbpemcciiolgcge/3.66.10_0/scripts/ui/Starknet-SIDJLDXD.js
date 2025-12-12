import{o as s,q as c}from"./chunk-SHG7TIBL.js";s();c();var u={getAccount:async({provider:t})=>{let[a]=await t?.starknet?.enable();return a},signMessage:async({provider:t,message:a})=>(await t?.starknet?.enable(),await t?.starknet?.account?.signMessage(a)),sendTransaction:async({provider:t,payload:a,cb:n,abi:r})=>{await t?.starknet?.enable();let{transaction_hash:e}=await t?.starknet?.account?.execute(a,r);return n&&n(e),e}},o=u;export{o as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=Starknet-SIDJLDXD.js.map
