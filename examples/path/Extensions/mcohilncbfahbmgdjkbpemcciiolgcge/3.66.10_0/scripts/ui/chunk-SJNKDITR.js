import{i as p}from"./chunk-ULXZERAJ.js";import{a as d}from"./chunk-532YJKK4.js";import{b as l}from"./chunk-OL67KS7C.js";import{f as h,o,q as s}from"./chunk-SHG7TIBL.js";o();s();function u(a){if(!a)return{};let{pathname:e,search:r,hash:n}=a,t=new URLSearchParams(r),c=new URLSearchParams(n.slice(1)),i=e?.includes("/dex-swap/meme/sell")||e?.includes("/dex-swap/meme")&&t?.get("tab")==="sell",P=e?.includes("/dex-swap/meme/buy")||e?.includes("/dex-swap/meme")&&t?.get("tab")==="buy";return{isSellMode:i,isBuyMode:P,path:e,queryParams:p(t),hashParams:c}}o();s();var m=h(l()),f=h(d());function L(){let a=(0,f.useLocation)(),[e,r]=(0,m.useState)({});(0,m.useEffect)(()=>{if(a){let i=u(a);r(i)}},[a]);let{path:n,queryParams:t,hashParams:c}=e;return{path:n,queryParams:t,hashParams:c}}export{u as a,L as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-SJNKDITR.js.map
