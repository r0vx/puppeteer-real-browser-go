import{f as p}from"./chunk-7JSJY3SQ.js";import{a as s}from"./chunk-CZZFLX67.js";import{o as M}from"./chunk-XP3TNE3R.js";import{jc as h}from"./chunk-QRHYK2TW.js";import{c as a,e as B}from"./chunk-AS4Z3FNS.js";import{f as c,o as n,q as i}from"./chunk-SHG7TIBL.js";n();i();var m=c(h()),u=c(M());B();var S=({metamask:t})=>t?.createdMap||{},g=(t,e,f={})=>{let r=p(t,e,{...f,withBalanceStatus:!0})||{},{requestBalance:l}=r,d=!(0,m.useSelector)(S)[e];return(0,u.useMount)(async()=>{if(d)try{let o=await a().getWalletTypeCreated(e);await a().createWalletToServer({walletId:e,walletType:o,noticeBackend:!0}),s(),l()}catch{}}),r},x=g;export{x as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-V2QGBG3D.js.map
