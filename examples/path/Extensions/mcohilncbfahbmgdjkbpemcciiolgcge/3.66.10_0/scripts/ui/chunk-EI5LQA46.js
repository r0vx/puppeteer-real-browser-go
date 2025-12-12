import{c as k}from"./chunk-FKISPHB5.js";import{l as d}from"./chunk-KDVER5QK.js";import{a as p}from"./chunk-PD44C7EF.js";import{jc as N}from"./chunk-QRHYK2TW.js";import{b as C}from"./chunk-OL67KS7C.js";import{f as m,o as f,q as a}from"./chunk-SHG7TIBL.js";f();a();var e=m(C()),b=m(N());var B=20*1e3,R=o=>{let c=(0,b.useDispatch)(),n=d(void 0,o),t=(0,e.useRef)(null),{extensionConfig:s}=p("rpc_info"),l=(0,e.useMemo)(()=>o||s,[o,s]),i=()=>{clearInterval(t.current),t.current=null},u=async()=>{try{let r=await n();if(!r?.getBlockNumber){i();return}let g=await r.getBlockNumber();c(k(g))}catch(r){console.log(`fetch block failed 
${r}`)}};(0,e.useEffect)(()=>(l?.rpcUrl&&(u(),t.current=setInterval(()=>{u()},B)),i),[l,n,c])},y=R;export{y as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-EI5LQA46.js.map
