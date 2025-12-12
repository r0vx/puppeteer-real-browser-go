import{Sb as o}from"./chunk-QRHYK2TW.js";import{T as i}from"./chunk-4EIUGT5A.js";import{o as a,q as n}from"./chunk-SHG7TIBL.js";a();n();function u(t){return t.metamask.domainMetadata||{}}function f(t){let s=u(t);return Object.values(s).reduce((e,{host:r})=>(r&&(e[r]?e[r]+=1:e[r]=1),e),{})}function m(t){return Object.values(t.metamask.pendingApprovals||{}).filter(e=>e.type===i.WALLET_REQUEST_PERMISSIONS).map(e=>({metadata:{id:e.id,origin:e.origin,url:e?.requestData?.url},permissions:{eth_accounts:{}},time:e?.time||Date.now(),providerType:e?.requestData?.providerType,providerTypes:e?.requestData?.providerTypes,method:e?.requestData?.method,exts:e?.requestData?.exts,providersExts:e?.requestData?.providersExts}))}function g(t){return(t.metamask.generalDappRequests||[]).map(e=>({...e,time:e?.timestamp}))}function q(t){return t.metamask.realGeneralDappRequestsCount||0}function p(t){let s=m(t);return s&&s[0]?s[s.length-1]:null}function R(t){let s=p(t);return s&&s.metadata?s.metadata.id:null}function x(t){let s=u(t),e=o(t);return{...s[e],origin:e}}export{u as a,f as b,m as c,g as d,q as e,p as f,R as g,x as h};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-F6UVHIJV.js.map
