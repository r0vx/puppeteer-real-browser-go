import{a as g}from"./chunk-TQWSBAXG.js";import{o as r,q as u}from"./chunk-SHG7TIBL.js";r();u();var i={getAccount:({provider:t})=>(t?.sui?.accounts?.[0]||{}).address,signMessage:async({provider:t,message:s})=>{let c=new Uint8Array(s.length);for(let n=0;n<s.length;n++)c[n]=s.charCodeAt(n);let a=i.getAccount({provider:t});return(await t?.sui?.features["sui:signMessage"].signMessage({message:c,account:a})||{}).signature},sendTransaction:async({provider:t,payload:s,cb:c})=>{let{TransactionBlock:a}=await import("./dist-MRCQ4IO7.js"),{transactionBlock:o}=s||{},n={};if(typeof o=="string"){let f=a.from(o),A=i.getAccount({provider:t});n={transactionBlock:f,account:A,chain:g.SUI}}else o instanceof a&&(n=s);let l=await t?.sui?.features["sui:signAndExecuteTransactionBlock"].signAndExecuteTransactionBlock(n),{digest:e}=l||{};return console.log(`sui send transaction success: ${e}`),c&&c(e),e}},B=i;export{B as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=Sui-Q5BOR6IV.js.map
