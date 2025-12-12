import{b as k,c as L,d as I,g as N,o as t,q as n}from"./chunk-SHG7TIBL.js";var E={};I(E,{EOL:()=>A,arch:()=>B,cpus:()=>d,default:()=>_,endianness:()=>p,freemem:()=>m,getNetworkInterfaces:()=>y,hostname:()=>s,loadavg:()=>l,networkInterfaces:()=>w,platform:()=>V,release:()=>v,tmpDir:()=>i,tmpdir:()=>b,totalmem:()=>x,type:()=>g,uptime:()=>c});function p(){if(typeof o>"u"){var r=new ArrayBuffer(2),u=new Uint8Array(r),f=new Uint16Array(r);if(u[0]=1,u[1]=2,f[0]===258)o="BE";else if(f[0]===513)o="LE";else throw new Error("unable to figure out endianess")}return o}function s(){return typeof globalThis.location<"u"?globalThis.location.hostname:""}function l(){return[]}function c(){return 0}function m(){return Number.MAX_VALUE}function x(){return Number.MAX_VALUE}function d(){return[]}function g(){return"Browser"}function v(){return typeof globalThis.navigator<"u"?globalThis.navigator.appVersion:""}function w(){}function y(){}function B(){return"javascript"}function V(){return"browser"}function i(){return"/tmp"}var o,b,A,_,h=k(()=>{t();n();b=i,A=`
`,_={EOL:A,tmpdir:b,tmpDir:i,networkInterfaces:w,getNetworkInterfaces:y,release:v,type:g,cpus:d,totalmem:x,freemem:m,uptime:c,loadavg:l,hostname:s,endianness:p}});var D=L((X,a)=>{t();n();var e=(h(),N(E));if(e&&e.default){a.exports=e.default;for(let r in e)a.exports[r]=e[r]}else e&&(a.exports=e)});export{D as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-K3QEQ66V.js.map
