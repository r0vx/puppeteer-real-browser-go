import{a as n}from"./chunk-7HRNNJBX.js";import{$ as u,Y as c,aa as i,i as r}from"./chunk-QMENNGDH.js";import{o as a,q as d}from"./chunk-SHG7TIBL.js";a();d();function R({isRpcMode:f,lastOptType:e,permissionsRequestId:o,isNotBackup:t}){if(f){if(e===n.connect&&!t)return`${r}/${o}`;if([n.msg,n.addToken,n.addChain,n.transaction].includes(e))return c;if(e===n.generalDappRequest)return i}else if(!t){if([n.transaction,n.addToken,n.addChain].includes(e))return u;if(e===n.connect)return`${r}/${o}`;if(e===n.generalDappRequest)return i}return""}export{R as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-BSCT2WLI.js.map
