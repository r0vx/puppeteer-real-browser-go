import{a as r,b as d,e,f as p,h as i,j as g,m as o,s as f}from"./chunk-QHXUL3YM.js";import{n as process,o as n,q as s}from"./chunk-SHG7TIBL.js";n();s();d();p();g();f();var c=i[process.env.ASSETS_BUILD_TYPE],a={countryCode:"",setCountryCode:t=>{a.countryCode=t,e.set({countryCode:t})}};(async()=>{let{countryCode:t}=await e.get("countryCode")||"";a.countryCode=t})();var l={environment:c,release:process.env.APP_VERSION,tracesSampleRate:.2,defaultIntegrations:!1,beforeSend:t=>(!t.exception||!t.exception.values||(t.tags||(t.tags={}),Array.isArray(t.exception.values)&&(t.exception.values[0].type=`${t.exception.values[0].type} ${process.env.APP_VERSION}`),t.tags.os=navigator.vendor||"Firefox",t.tags.browserName=navigator.language,t.tags.language=navigator.language,t.tags.device=navigator.product,t.tags.browser=navigator.userAgent,t.tags.environment=c,t.tags.countryCode=a.countryCode),t),hideBreadcrumb:!0,noDefaultReport:!0,sendClientReports:!1},u=`https://b25a7d61802a41b6bf564e64acff1b50@${o.getUrl()}/apmfe/120`;async function y(){{let{devid:t}=await e.get("devid");await o.setUrlFromLocalAsync(),u=`https://b25a7d61802a41b6bf564e64acff1b50@${o.getUrl()}/apmfe/120`,r.init({dsn:u,devId:t,...l})}return r}export{l as a,u as b,y as c};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-WKF2ZHWH.js.map
