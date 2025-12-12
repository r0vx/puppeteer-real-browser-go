import{o as i,q as c}from"./chunk-SHG7TIBL.js";i();c();var g={getAccount:async({provider:t})=>{let{address:n}=await t?.bitcoinSignet?.connect();return n},getPublicKey:async({provider:t})=>{let{publicKey:n}=await t?.bitcoinSignet?.connect();return n},signMessage:async({provider:t,message:n,address:e})=>await t?.bitcoinSignet?.signMessage(n,{from:e}),signPsbt:async({provider:t,psbtHex:n,account:e,type:s,...a})=>await t?.bitcoinSignet?.signPsbt(n,{from:e,type:s,...a})},b=g;export{b as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=BtcSignet-TLOIPEQJ.js.map
