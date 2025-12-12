import{o,q as i}from"./chunk-SHG7TIBL.js";o();i();var u={getAccount:async({provider:n})=>n?.tronWeb?.defaultAddress?.base58,signMessage:({provider:n,message:e})=>new Promise((a,t)=>{let{tronWeb:c}=n||{};c?.trx?.signMessage(e).then(s=>{a(s)}).catch(s=>{t(s)})}),sendTransaction:async({provider:n,payload:e,cb:a})=>{try{let{tronWeb:t}=n||{},c=await t?.trx?.sign(e),s=await t?.trx?.sendRawTransaction(c),{transaction:{txID:r}}=s||{};return a&&a(r),r}catch(t){throw t}}},g=u;export{g as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=Tron-KFDK5BNC.js.map
