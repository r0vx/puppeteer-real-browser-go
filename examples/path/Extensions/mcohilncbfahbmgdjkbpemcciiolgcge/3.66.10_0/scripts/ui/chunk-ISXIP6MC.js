import{g}from"./chunk-XZE3L7K7.js";import{p as h,q as b}from"./chunk-VL2YBSHB.js";import{Xa as m,gc as f,ib as T,ic as u,od as P}from"./chunk-QHXUL3YM.js";import{a as K}from"./chunk-LMBQGBLO.js";import{ra as A,y as d}from"./chunk-CYTB2B6Q.js";import{o as w,q as l}from"./chunk-SHG7TIBL.js";w();l();T();K();P();A();var B=(o,t)=>async(a,e,c)=>{let r=`0/${a}`,{extendedPublicKey:s}=d(c,{path:t})||{},{hardwareDerivePubKey:i,getAddressByPublicKey:p}=await m(),n=await i(s,r),y=await p(0,{publicKey:n,addressType:g[o]});e[u][o]={path:`${t}/${r}`,publicKey:n,address:y}},M=async(o,t,a)=>{t[u]={};for(let e=0;e<b.length;e++){let{type:c,basePath:r}=b[e];await B(c,r)(o,t,a)}},S=(o,t)=>async(a,e,c)=>{let r=t+a,{extendedPublicKey:s}=d(c,{path:h})||{},{hardwareDerivePubKey:i,getAddressByPublicKey:p}=await m(),n=await i(s,r),y=await p(60,{publicKey:n});e[f][o]={path:`${h}/${r}`,address:y}};export{M as a,S as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-ISXIP6MC.js.map
