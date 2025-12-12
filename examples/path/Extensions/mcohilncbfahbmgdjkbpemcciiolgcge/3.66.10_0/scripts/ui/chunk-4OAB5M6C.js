import{d as M}from"./chunk-7JSJY3SQ.js";import{b as B}from"./chunk-7C7MBE2C.js";import{Ab as g,Gd as A,Jb as G,K as u,Ma as C,Na as w,Pd as N,ia as q}from"./chunk-QHXUL3YM.js";import{L as r,ra as k}from"./chunk-CYTB2B6Q.js";import{b as T}from"./chunk-OL67KS7C.js";import{f as b,o as f,q as p}from"./chunk-SHG7TIBL.js";f();p();var t=b(T());k();N();w();G();q();var O=({accountId:n,chainIndex:o})=>{let[e,d]=(0,t.useState)({}),[E,i]=(0,t.useState)(!1),[L,c]=(0,t.useState)(!1),S=A({chainId:o}),v=M(S?.coinId)?.decimals,a=(0,t.useCallback)(async()=>{try{i(!0),c(!1);let{data:s}=await g(C.getAptosBaseCoinBalance,{accountId:n,chainIndex:o});d(y=>({...y,[o]:s.coinAmountOrigin}))}catch{e[o]||c(!0)}finally{i(!1)}},[o,e,n]);(0,t.useEffect)(()=>{r(o)||a()},[o]);let m=B("wallet-asset",{pollingInterval:10*1e3});(0,t.useEffect)(()=>{let s=m?.data?.aptMCAssetChanged;!r(o)&&s&&a()},[m,o]);let l=e[o];return{requestBalance:a,isBalanceLoading:E,isBalanceLoadError:L,coinAmountInt:l,coinAmount:u(l,10**v)}},R=O;export{R as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-4OAB5M6C.js.map
