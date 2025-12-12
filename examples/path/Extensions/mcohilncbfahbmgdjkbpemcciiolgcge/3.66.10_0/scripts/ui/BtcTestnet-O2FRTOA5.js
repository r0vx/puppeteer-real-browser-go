import{o as c,q as i}from"./chunk-SHG7TIBL.js";c();i();var o={getAccount:async({provider:t})=>{let{address:n}=await t?.bitcoinTestnet?.connect();return n},getPublicKey:async({provider:t})=>{let{publicKey:n}=await t?.bitcoinTestnet?.connect();return n},signMessage:async({provider:t,message:n,address:e})=>await t?.bitcoinTestnet?.signMessage(n,{from:e}),signPsbt:async({provider:t,psbtHex:n,account:e,type:s,...a})=>await t?.bitcoinTestnet?.signPsbt(n,{from:e,type:s,...a})},b=o;export{b as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=BtcTestnet-O2FRTOA5.js.map
