import{Ab as c,Eb as n,Jb as i,Ma as a,Na as s}from"./chunk-QHXUL3YM.js";import{o as r,q as o}from"./chunk-SHG7TIBL.js";r();o();s();i();var A=async(t={})=>{let{data:e}=await c(a.queryAccountExist,t);return e},p=async t=>{let{data:e}=await c(a.queryAccountInfo,t);return e},m=async(t,e)=>await n(a.createWaxAccount,t,{walletSignParams:{needWalletSign:!0,walletId:e}})||{},w=async(t,e)=>await n(a.createFreeWaxAccount,t,{walletSignParams:{needWalletSign:!0,walletId:e}})||{},W=async t=>{let{data:e}=await c(a.queryAccountStatus,t);return e||{}},d=async t=>{let{data:e}=await c(a.checkAccountPattern,t);return e??!1};export{A as a,p as b,m as c,w as d,W as e,d as f};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-FBGIZDXX.js.map
