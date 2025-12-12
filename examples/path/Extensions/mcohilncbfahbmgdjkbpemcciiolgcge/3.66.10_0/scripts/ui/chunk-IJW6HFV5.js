import{a as d}from"./chunk-7UDMBSMI.js";import{c as a,f as y}from"./chunk-XZE3L7K7.js";import{H as I}from"./chunk-QRHYK2TW.js";import{Nc as c,Wc as f,ee as m,fe as u,od as g,oe as P}from"./chunk-QHXUL3YM.js";import{o as p,q as C}from"./chunk-SHG7TIBL.js";p();C();P();g();function B({coin:i,walletIdentity:e,options:o={}}){let{needFilterBaseCoin:t=!1,isKeystone:n,isMPC:r,isHardWallet:s}=o,W=n??a(e?.initialType),F=s??y(e?.keyringIdentityType),l=r??d(e?.keyringIdentityType);return!l&&!F?!0:t&&I(i)?!!l:W&&i.baseCoinId===c&&i.coinId===f?!1:l?!Object.values(u).includes(i.protocolId):!Object.values(m).includes(i.protocolId)}function O({coins:i=[],walletIdentity:e,options:o={}}){let t=a(e?.initialType),n=y(e?.keyringIdentityType),r=d(e?.keyringIdentityType);return i.filter(s=>B({coin:s,walletIdentity:e,options:{...o,isMPC:r,isHardWallet:n,isKeystone:t}}))}export{B as a,O as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-IJW6HFV5.js.map
