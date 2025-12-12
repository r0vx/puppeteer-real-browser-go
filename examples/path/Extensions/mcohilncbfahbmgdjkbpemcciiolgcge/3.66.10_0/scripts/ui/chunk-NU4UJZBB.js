import{a as g}from"./chunk-2FDQIA7D.js";import{c as a}from"./chunk-L524QXHA.js";import{E as p}from"./chunk-4EIUGT5A.js";import{Gd as m,Id as f,Pd as S,Wa as E,gb as I,hb as s,ib as N}from"./chunk-QHXUL3YM.js";import{c as h,i as u,ra as l}from"./chunk-CYTB2B6Q.js";import{f as k,o as d,q as T}from"./chunk-SHG7TIBL.js";d();T();var r=k(E());l();N();S();var W=async o=>{let{Common:t,Hardfork:e}=await I();(0,r.isHexString)(u(o.chainId))&&(o.chainId=h(g(o.chainId)));let n=m({netWorkId:o.chainId})?.baseChain,i=()=>{let w=a(o.from,n),b=a(o.to,n);return{...o,from:w,to:b,gasLimit:o.gas||o.gasLimit}},c=m({netWorkId:o.chainId})?.localType||"custom-net",x=f(c)?.networkId||"custom-net",y={chainId:o.chainId,networkId:x,name:c},C={common:t.custom(y,{baseChain:n,hardfork:e.London})},{TransactionFactory:A}=await s();return A.fromTxData(i(),C)},_=async(o,t)=>{let e=o.toJSON();e.type=o.type;let{TransactionFactory:n}=await s(),i=n.fromTxData({...e,...t},{common:o.common,freeze:Object.isFrozen(o)});return(0,r.bufferToHex)(i.serialize())},D="0x2019",j=({chainId:o,method:t})=>D===o&&t===p.KAIA_SIGN_TRANSACTION;export{W as a,_ as b,j as c};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-NU4UJZBB.js.map
