import{o as c}from"./chunk-QRHYK2TW.js";import{Wa as P,Xa as o,ib as h,ra as m}from"./chunk-QHXUL3YM.js";import{f as d,o as s,q as n}from"./chunk-SHG7TIBL.js";s();n();var i=d(P());h();m();var x=async(t,e,r,a)=>{try{return await a(t,{privateKey:e,hrp:r}),!0}catch{return!1}},v=async(t,e)=>{let r=[],a=c(e),{getNewAddress:f}=await o();return await Promise.all(a.map(({coinType:p,cosmosPrefix:l,baseChain:u})=>x(p,t,l,f).then(y=>{y&&r.push(u)}))),r};var B=async(t,e)=>await v(t,e),K=async(t,e)=>{let r=await B(t,e);return Boolean(r[0])};export{B as a,K as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-YQ5FMCKJ.js.map
