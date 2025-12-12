import{a as L}from"./chunk-VIT5MDY4.js";import{c as e,d as o}from"./chunk-RH6WPBEZ.js";import{h as f,n as l}from"./chunk-AZNX5TUC.js";import{Zd as A,oe as _}from"./chunk-QHXUL3YM.js";import{o as S}from"./chunk-XR4SQWL7.js";import{b as C}from"./chunk-OL67KS7C.js";import{f as T,o as n,q as c}from"./chunk-SHG7TIBL.js";n();c();n();c();var E=T(C());var w=({keyringId:u="",actionMap:r,customConfig:i})=>{let a=l(u),{accountStatus:t,mpcStatus:s}=a?.status||{},M=(0,E.useMemo)(()=>t===e.FROZEN||t===e.DELETE||s===o.ESCAPE||s===o.RECOVER,[t,s]),W=(0,E.useMemo)(()=>{let p,m;switch(!0){case s===o.RECOVER:p=r["RECOVER"],m=r["SWITCH_WALLET"];break;case s===o.ESCAPE:case t===e.FROZEN:case t===e.DELETE:p=r["SWITCH_WALLET"];break;default:break}return L({accountStatus:t,mpcStatus:s},{onConfirm:p,onCancel:m,...i})},[t,r,s]);return E.default.createElement(S.Prompt,{visible:M,...W})};n();c();_();var P=T(C()),K=u=>{let r=f(u),i=(0,P.useMemo)(()=>{let{accountStatus:a,mpcStatus:t}=r?.status||{accountStatus:e.NORMAL,mpcStatus:o.INIT};switch(!0){case t===o.RECOVER:case t===o.ESCAPE:case a===e.FROZEN:case a===e.DELETE:return!1;default:return!0}},[r]);return r?.keyringIdentityType!==A.MPC||i};export{w as a,K as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-Q4BJZ4WV.js.map
