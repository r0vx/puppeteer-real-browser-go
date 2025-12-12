import{Ra as r,Sa as E}from"./chunk-QHXUL3YM.js";import{b as l}from"./chunk-OL67KS7C.js";import{f as y,o,q as s}from"./chunk-SHG7TIBL.js";o();s();var i=y(l());E();var m=n=>{let[a,c]=(0,i.useState)({});return(0,i.useEffect)(()=>{let e;return(async()=>{let t=await Promise.resolve(r.extension_config),f=await t.get(n);c(f||{}),e=t.liveQuery({extensionKey:n}).subscribe((p,x)=>{!x&&p?.length&&c(p[0])})})(),()=>{e&&e?.unsubscribe()}},[n]),{extensionConfig:a,setExtensionConfig:async e=>{try{await(await Promise.resolve(r.extension_config)).set({...e,extensionKey:n})}catch{console.log("setRpcInfo fail")}}}};o();s();export{m as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-PD44C7EF.js.map
