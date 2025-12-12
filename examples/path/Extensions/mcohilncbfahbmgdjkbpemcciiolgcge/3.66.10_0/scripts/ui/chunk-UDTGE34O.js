import{e as g}from"./chunk-OK4242CI.js";import{c as A,i as T}from"./chunk-IEGAVAV7.js";import{g as I}from"./chunk-AZNX5TUC.js";import{o as D}from"./chunk-XP3TNE3R.js";import{Td as y,Xd as S,md as f,od as L}from"./chunk-QHXUL3YM.js";import{o as l,ra as b}from"./chunk-CYTB2B6Q.js";import{f as w,o as d,q as m}from"./chunk-SHG7TIBL.js";d();m();var i=w(D());b();L();S();var v=(r,C)=>{let h=I(),s=C??h,u=g(s),a=(0,i.useCreation)(()=>u.find(n=>n.coinId===r?.coinId),[u,r?.coinId])?.childrenCoin??[],o=A(r?.localType,s),c=T(r?.localType,s);return(0,i.useCreation)(()=>{if(!r||!Array.isArray(a)||!Array.isArray(o)||!o.length)return[];let n=a.filter(e=>e.coinId===+r?.coinId).map(e=>({...e})),p=[],t=l(n[0]||r),B=n.map(e=>c[e.addressType]);return o.forEach(({address:e,addressType:W})=>{B.includes(e)||(t.address=e,t.userAddress=e,t.addressType=y[f(r?.localType)]?.[W],t.coinAmount=0,t.coinAmountInt=0,t.currencyAmount=0,t.currencyAmountUSD=0,p.push(l(t)))}),n.concat(p).filter(e=>Boolean(c[e.addressType]))},[r,a,o,c])};export{v as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-UDTGE34O.js.map
