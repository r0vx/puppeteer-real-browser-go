import{a as h}from"./chunk-J3K74NCM.js";import{Gd as P,Pd as R,Xa as f,bb as y,ib as M}from"./chunk-QHXUL3YM.js";import{H as K,y as p}from"./chunk-MOTAOJVG.js";import{H as L}from"./chunk-XR4SQWL7.js";import{b as E}from"./chunk-OL67KS7C.js";import{f as k,o as g,q as u}from"./chunk-SHG7TIBL.js";g();u();var t=k(E());K();M();R();function D({fromAddress:a,disabled:n,localType:w,inputData:m,createInputDataFn:i,upgradeAddress:s,isUpgrade7702:c}){let e=P({localType:w}),[x,r]=(0,t.useState)(!1),[v,G]=(0,t.useState)(null),l=e?.realChainIdHex;return(0,t.useEffect)(()=>{if(n){r(!1);return}r(!0),(async()=>{try{let{getRandomPrivateKey:z,getNewAddress:_}=await f(),o=await z(e?.coinType),{address:b}=await _(e?.coinType,{privateKey:o}),A=i?await i({address:b,privateKey:o}):m,d={address:a,tokenAddress:a,value:"0",inputData:A,authorizationList:[],stateOverrideList:[{address:a,code:c?`0xef0100${s.substring(2).toLowerCase()}`:"0x"}]},C={privateKey:o,data:{chainId:l,address:s,nonce:h(1)}},{EthWallet:I}=await y(),T=await new I().signAuthorizationListItemForRPC(C);d.authorizationList=[T],G(d),r(!1)}catch{L.error({title:p("wallet_extension_general_error_extension_code_error")})}})()},[a,n,m,i,l,s,e?.coinType,c]),{gasLimitParams:v,loading:x}}var $=D;export{$ as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-OAQKAWH5.js.map
