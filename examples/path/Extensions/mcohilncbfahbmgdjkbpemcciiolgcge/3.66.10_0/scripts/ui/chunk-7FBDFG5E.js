import{o as _}from"./chunk-XP3TNE3R.js";import{V as D}from"./chunk-QRHYK2TW.js";import{e as d,ia as n,ra as B}from"./chunk-CYTB2B6Q.js";import{b as T}from"./chunk-OL67KS7C.js";import{f as b,o as v,q as y}from"./chunk-SHG7TIBL.js";v();y();var e=b(T()),s=b(_());B();var j=(E,g={wait:200,disabled:!1,fetchOnce:null,forceUpdate:null,onFetchSuccess:()=>{},onFetchError:()=>{}})=>{let[w,l]=(0,e.useState)({}),[m,S]=(0,e.useState)(null),[U,{setTrue:k,setFalse:r}]=(0,s.useBoolean)(!0),[q,{setFalse:i}]=(0,s.useBoolean)(!0),{address:o,inputData:a,tokenAddress:c,coinId:h,value:F,authorizationList:p,stateOverrideList:L}=E,{wait:A,disabled:G,fetchOnce:t,forceUpdate:P,onFetchSuccess:I,onFetchError:O}=g,x=async()=>{try{let u={coinId:h,value:F,address:o&&n(o),inputData:a&&n(a),authorizationList:p,stateOverrideList:L};c&&(u.tokenAddress=n(c));let{data:f}=await D(u);l(f),t&&S(t),d(I)&&I()}catch{l(f=>({...f,queryGasLimitErrorUseDefault:!0})),d(O)&&O()}finally{r(),i()}},{run:z}=(0,s.useDebounceFn)(()=>{if(G){r(),i();return}if(t===m&&t!==null){r(),i();return}x()},{wait:A});return(0,e.useEffect)(()=>{k(),z()},[o,a,c,p,L,h,F,t,P,m,G]),[w,U,q]},K=j;export{K as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-7FBDFG5E.js.map
