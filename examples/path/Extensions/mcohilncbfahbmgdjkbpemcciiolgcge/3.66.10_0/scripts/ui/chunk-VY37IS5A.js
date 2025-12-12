import{t as u}from"./chunk-NKIUEXOA.js";import{o as x}from"./chunk-XP3TNE3R.js";import{da as r}from"./chunk-4EIUGT5A.js";import{S as m,ia as w}from"./chunk-QHXUL3YM.js";import{H as E,y as o}from"./chunk-MOTAOJVG.js";import{f as h,o as s,q as n}from"./chunk-SHG7TIBL.js";s();n();var c=h(x());E();w();function y(){let l=u();return(0,c.useMemoizedFn)(async({from:L,chainId:f,simulateTransactionParam:p={},...T})=>{let e=(await l({checkTypes:[r.TX_ANALYZE],from:L,chainId:f,bizLine:6,simulateTransactionParamList:[{sigVerify:!1,replaceRecentBlockhash:!0,...p}],...T}))?.[r.TX_ANALYZE]||{},[a]=e.simulateTransactionResultList||[],i=(e.simulateTransactionResultList||[]).find(t=>t?.msg||m(t?.unitGasLimit,"0"));if(i?.msg)throw new Error(i?.msg);if(!a||!!i)throw new Error(o("wallet_extension_alert_estimate_unavailable"));return{firstUnitLimit:a?.unitGasLimit,unitLimits:(e.simulateTransactionResultList||[]).map(t=>t?.unitGasLimit)}})}var G=y;export{G as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-VY37IS5A.js.map
