import{b as y}from"./chunk-YGIVEMKT.js";import{Wa as f,hc as c,od as x}from"./chunk-QHXUL3YM.js";import{f as o,o as p,p as s,q as u}from"./chunk-SHG7TIBL.js";p();u();var r=o(f()),i=o(y());x();var b=(t,e)=>{let n="";try{n=i.default.encodeMethod({constant:!1,inputs:[{name:"_to",type:"address"},{name:"_value",type:"uint256"}],name:"transfer",outputs:[{name:"",type:"bool"}],payable:!1,stateMutability:"nonpayable",type:"function"},[(0,r.addHexPrefix)(t),e])}catch(a){console.log(a)}return n};function g(t,e){return t.type===c?t.uniqueId===e.uniqueId:t.chainId===e.realChainIdHex&&t.type===e.localType}var h=t=>{try{let e=(0,r.stripHexPrefix)(t),n=s.Buffer.from(e,"hex"),a=n.length===32?t:n.toString("utf8"),l="[\0-\b\v\f-\x7F\x80-\x9F]";return new RegExp(l).test(a)?t:a}catch{return t}};export{b as a,g as b,h as c};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-36PYO4PJ.js.map
