import{o as t,q as r}from"./chunk-SHG7TIBL.js";t();r();t();r();var S="content_script_message",_={goToDexSwapMemeMode:"goToDexSwapMemeMode",notifyRedirect:"notifyRedirect"},n="redirect_param_",l="redirect_pathname",R="redirect_search_params";var M=a=>Object.entries(a).map(([e,o])=>`${e}=${encodeURIComponent(o)}`).join("&"),C=a=>{let e=new URLSearchParams(a),o=e.get(l),s=e.get(R),i={};return s&&s.split("&").filter(c=>c.includes(n)).forEach(c=>{let[m,p]=c.split("="),d=m.replace(n,"");i[d]=decodeURIComponent(p)}),{redirectPathname:o,redirectSearchParams:i}};export{S as a,_ as b,M as c,C as d};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-LU2QCUPE.js.map
