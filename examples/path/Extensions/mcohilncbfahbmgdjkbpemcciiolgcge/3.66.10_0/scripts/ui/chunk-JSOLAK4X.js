import{d as g}from"./chunk-AADHEW4J.js";import{a as x,b as S,c as M,d as N}from"./chunk-XZE3L7K7.js";import{zb as T}from"./chunk-QRHYK2TW.js";import{c as o,e as O}from"./chunk-AS4Z3FNS.js";import{z as I}from"./chunk-4EIUGT5A.js";import{u as W}from"./chunk-VL2YBSHB.js";import{I as P,Pd as B,la as w,pd as m,ra as E,rd as y,t as u}from"./chunk-QHXUL3YM.js";import{o as l,q as p}from"./chunk-SHG7TIBL.js";l();p();P();E();var H=()=>w()===u,f=t=>x(t)||M(t)||N(t)||S(t);l();p();B();O();var Z=({txData:t,txParams:n,walletId:e,isRpcMode:r=!1,baseChain:i=m})=>async(s,a)=>{let c=a();e??=T(c);let d=await o().getWalletIdentityByWalletId(e);f(d?.initialType)&&await g({walletInfo:d,txData:t,txParams:n,isRpcMode:r,baseChain:i})};async function A(t,n,e,r,{...i}={}){let s="";r??=await o().getWalletIdByAddress(n,e);let a=await o().getWalletIdentityByWalletId(r);try{if(f(a?.initialType))return s=await g({walletInfo:a,txParams:t,baseChain:e}),s;s=await o().signTransaction(t,n,e,r,i)}catch(c){throw c?.message===W?c:new Error(I)}return s}function $(t,n,e){return async()=>A(t,n,y,e)}function C(t,n,e,r,i){return o().signPsbt(t,n,e,r,i)}export{H as a,f as b,Z as c,A as d,$ as e,C as f};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-JSOLAK4X.js.map
