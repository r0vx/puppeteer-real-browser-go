import{b as r}from"./chunk-RJ3YTAFM.js";import{a as u}from"./chunk-OCAFSBNZ.js";import{k as h,l as i,n as d}from"./chunk-XR4SQWL7.js";import{D as c,ra as k}from"./chunk-CYTB2B6Q.js";import{b as p}from"./chunk-OL67KS7C.js";import{f as T,o as n,q as o}from"./chunk-SHG7TIBL.js";n();o();var a=T(p());k();function s(){return window.matchMedia("(prefers-color-scheme: dark)").matches?r.DARK:r.LIGHT}function f(t,e){return e==="auto"&&(e=s()),c(t)?t.replace(/-light\.|-dark\./g,`-${e}.`):c(t[e])&&t[e].match(/^\/?cdn\/assets\//)?`${u()}${t[e]}`:t[e]}function l(){return window.localStorage.getItem("theme")||"auto"}function I(){let[t,e]=(0,a.useState)("auto");return(0,a.useEffect)(()=>{e(l())},[]),t}function x(t){let e=t==="auto"?s():t;i(e),window.localStorage.setItem("theme",t)}function L(t){let e=d()||s();return f(t,e)}function O(t){let e=h()||s();return f(t,e)}function P(){let t=l();if(t===r.AUTO){let e=window.matchMedia("(prefers-color-scheme: dark)"),m=window.matchMedia("(prefers-color-scheme: light)");x("auto"),e.addEventListener("change",({matches:g})=>{g&&i(r.DARK)}),m.addEventListener("change",({matches:g})=>{g&&i(r.LIGHT)})}else i(t)}n();o();export{l as a,I as b,x as c,L as d,O as e,P as f};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-AVC4ODUV.js.map
