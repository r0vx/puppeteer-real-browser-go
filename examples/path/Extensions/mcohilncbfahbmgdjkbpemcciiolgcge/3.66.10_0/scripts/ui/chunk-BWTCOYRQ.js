import{Ab as T,Jb as d,Ma as f,Na as p,Ta as r,Ua as E}from"./chunk-QHXUL3YM.js";import{H as m,o}from"./chunk-XR4SQWL7.js";import{o as i,q as s}from"./chunk-SHG7TIBL.js";i();s();E();p();d();var g=async({confirmText:a,callback:e})=>{try{await T(f.getDisabedCreateAndImport),e&&e()}catch(t){t.status===r.STATUS_CODE.ERR_NETWORK||t.status===r.STATUS_CODE.ERR_TIMEOUT?e&&e():t.code=="900003"?o.tip({infoType:o.Tip.INFO_TYPE.default,title:t.msg,confirmText:a,onConfirm:n=>{n.destroy()}}):m.error(`System error, ${t.status} ${t.msg}`)}},R=g;export{R as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-BWTCOYRQ.js.map
