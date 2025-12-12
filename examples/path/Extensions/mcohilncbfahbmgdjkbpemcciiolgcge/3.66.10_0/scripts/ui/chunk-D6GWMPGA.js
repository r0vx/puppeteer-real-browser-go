import{a as T,k as E}from"./chunk-AZNX5TUC.js";import{q as l,r as m}from"./chunk-QMENNGDH.js";import{a as L}from"./chunk-532YJKK4.js";import{c as p,d,g as c}from"./chunk-COS5CL3D.js";import{e as i}from"./chunk-VL2YBSHB.js";import{b as y}from"./chunk-OL67KS7C.js";import{f as o,o as a,q as s}from"./chunk-SHG7TIBL.js";a();s();var f=o(T()),u=o(L());var C=o(y());var _=()=>{let e=(0,u.useHistory)(),t=E();return(0,C.useCallback)(async r=>{let g=await c.hasConnectedLedger(),{walletName:h}=t(r),n=`${m}?${f.default.stringify({type:i.addChain,walletId:r})}`;g?e.push(n):d.openModal(p.hardWareNotConnected,{walletName:h,onButtonClick:()=>{globalThis.platform.openExtensionInBrowser(l)},onExtButtonClick:()=>{globalThis.platform.openExtensionInBrowser(`${n}&hideBack=1`)}})},[e,t])};export{_ as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-D6GWMPGA.js.map
