import{c as d,o as a,q as i}from"./chunk-SHG7TIBL.js";var y=d((m,l)=>{a();i();l.exports=p;function n(){return new DOMException("The request is not allowed","NotAllowedError")}async function s(t){if(!navigator.clipboard)throw n();return navigator.clipboard.writeText(t)}async function w(t){let e=document.createElement("span");e.textContent=t,e.style.whiteSpace="pre",e.style.webkitUserSelect="auto",e.style.userSelect="all",document.body.appendChild(e);let o=window.getSelection(),c=window.document.createRange();o.removeAllRanges(),c.selectNode(e),o.addRange(c);let r=!1;try{r=window.document.execCommand("copy")}finally{o.removeAllRanges(),window.document.body.removeChild(e)}if(!r)throw n()}async function p(t){try{await s(t)}catch(e){try{await w(t)}catch(o){throw o||e||n()}}}});export{y as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-APGT6UVG.js.map
