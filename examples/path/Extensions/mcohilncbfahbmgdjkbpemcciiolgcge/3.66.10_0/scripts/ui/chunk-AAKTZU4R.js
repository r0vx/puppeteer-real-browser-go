import{a as b}from"./chunk-532YJKK4.js";import{k as d,l as C}from"./chunk-ZDOH3CEJ.js";import{b as P}from"./chunk-OL67KS7C.js";import{f as u,o as s,q as o}from"./chunk-SHG7TIBL.js";s();o();var f=u(P()),i=u(b()),B=()=>{let e=(0,i.useHistory)(),t=(0,i.useLocation)();return(0,f.useCallback)((r,l={})=>{let{overrideCurrentPath:n=!1,clearSearchParams:a=!1,preserveState:h=!0,replaceInsteadOfPush:v=!1,removeExistingSearchKeys:g=[],newSearchParams:S=""}=l,{search:y}=t,m=new URLSearchParams(y);g&&g.forEach(L=>{m.delete(L)}),e.replace({pathname:n||t.pathname,search:a?"":m.toString(),state:h?t.state:null}),e[v?"replace":"push"]({pathname:r,search:S||""})},[e,t])},w=B;s();o();C();var k={light:"light",dark:"dark"},p=d({name:"activityBanners",initialState:{bannerList:[]},reducers:{setBannerList:(e,t)=>{let{allData:c,closedBanners:r,theme:l}=t.payload,n=c;r?.length>0&&(n=c.filter(a=>!r.find(h=>h===a.id))),e.bannerList=n.map(a=>({...a,img:l===k.dark?a.nightLogo:a.dayLogo}))}}}),{setBannerList:I}=p.actions,K=p.reducer,M=e=>e[p.name].bannerList;export{I as a,K as b,M as c,w as d};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-AAKTZU4R.js.map
