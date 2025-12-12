import{o as h}from"./chunk-XP3TNE3R.js";import{b}from"./chunk-OL67KS7C.js";import{f as P,o as y,q as A}from"./chunk-SHG7TIBL.js";y();A();var o=P(b()),n=P(h());function H(v,p){let[i,d]=(0,o.useState)(!1),[L,l]=(0,o.useState)("unknown"),[u,f]=(0,o.useState)(!1),s=(0,n.useMemoizedFn)(()=>{i||(d(!0),p?.onActivate?.())}),r=(0,n.useMemoizedFn)(()=>{i&&(d(!1),p?.onDeactivate?.())}),c=(0,n.useMemoizedFn)(()=>{u||f(!0)});(0,n.useClickAway)(()=>{r()},v),(0,o.useEffect)(()=>{let e=v.current;if(!e)return;let m=t=>{t.pointerType==="mouse"&&(l(t.pointerType),s())},T=t=>{t.pointerType==="mouse"&&r()},a=t=>{(t.pointerType==="touch"||t.pointerType==="pen")&&(l(t.pointerType),c(),i?r():s())},E=t=>{t.pointerType==="mouse"&&c()};return e.addEventListener("pointermove",E),e.addEventListener("pointerenter",m),e.addEventListener("pointerleave",T),e.addEventListener("pointerdown",a),()=>{e.removeEventListener("pointermove",E),e.removeEventListener("pointerleave",T),e.removeEventListener("pointerdown",a),e.removeEventListener("pointerdown",a)}},[v,i,s,r,c]);let M=(0,n.useMemoizedFn)(e=>{(e?s:r)()}),w=(0,n.useMemoizedFn)(()=>{f(!1)});return{isActive:i,pointerType:L,setActive:M,moveActive:u,resetMoved:w}}export{H as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-OZT6PAEJ.js.map
