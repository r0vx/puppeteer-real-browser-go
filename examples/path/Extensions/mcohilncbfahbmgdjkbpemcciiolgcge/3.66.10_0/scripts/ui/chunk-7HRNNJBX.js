import{c as g,d as m,e as R}from"./chunk-F6UVHIJV.js";import{d}from"./chunk-QYDRYMWS.js";import{Nb as r,Ob as p,ub as a,wb as l,xb as c}from"./chunk-QRHYK2TW.js";import{o as i,q as s}from"./chunk-SHG7TIBL.js";i();s();var n={connect:"connect",transaction:"transaction",msg:"msg",addToken:"addToken",addChain:"addChain",generalDappRequest:"generalDappRequest"};function u(e={}){let t=null,o=0;return Object.keys(e).forEach(f=>{let T=e[f];Array.isArray(T)&&T.forEach(q=>{let h=q.time||0;h>=o&&(t=f,o=h)})}),t}i();s();var U=e=>{let t=d(e)?.isRpcMode,o;return t?o=u({[n.generalDappRequest]:m(e),[n.connect]:g(e),[n.transaction]:l(e),[n.msg]:a(e),[n.addToken]:p(e),[n.addChain]:r(e)}):o=u({[n.generalDappRequest]:m(e),[n.connect]:g(e),[n.transaction]:c(e),[n.addToken]:p(e),[n.addChain]:r(e)}),o},A=e=>{let t=d(e)?.isRpcMode,o=R(e);return t?g(e).length+l(e).length+a(e).length+p(e).length+r(e).length+o:g(e).length+c(e).length+p(e).length+r(e).length+o};export{n as a,U as b,A as c};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-7HRNNJBX.js.map
