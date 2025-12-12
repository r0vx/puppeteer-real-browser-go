import{_ as d}from"./chunk-POP3LIKS.js";import{j as e,la as P,ra as p}from"./chunk-CYTB2B6Q.js";import{o as m,q as I}from"./chunk-SHG7TIBL.js";m();I();p();var a=class{constructor(){this.getDefiPlatformIds=(o,t,f)=>o?Array.isArray(f)?P(f.map(r=>r.bridge?.defiPlatformId).filter(r=>r!==void 0)):[d]:e(t,"defiPlatformInfoList",[{defiPlatformId:d}]).map(r=>r.defiPlatformId);this.getCurrentRouteDefiPlatformId=(o,t)=>o?e(t,"bridge.defiPlatformId"):e(t,"defiPlatformId",d);this.getApproveInfo=(o,t)=>{if(o){let r=e(t,"quote.bestRoute"),i=e(t,"quote.bestRoute.pathSelectionRouterList",[]),n={};return Array.isArray(i)&&i.forEach(l=>{let u=l?.bridge?.defiPlatformId,s=l?.bridge?.dexMultiTokenAllowanceOut;u&&s&&(n[u]=s)}),{needApprove:e(t,"quote.bestRoute.needApprove"),defiPlatformId:e(r,"bridge.defiPlatformId"),defiPlatformIds:this.getDefiPlatformIds(o,r,i),allowanceData:n}}let f=e(t,"quote.bestRoute");return{needApprove:e(f,"needApprove"),defiPlatformId:e(f,"defiPlatformId",d),defiPlatformIds:this.getDefiPlatformIds(!1,f)}}}},A=new a;export{A as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-INQKED5D.js.map
