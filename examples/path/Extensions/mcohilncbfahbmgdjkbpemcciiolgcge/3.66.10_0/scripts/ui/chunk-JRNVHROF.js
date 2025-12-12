import{a as l}from"./chunk-ZZH7FUZX.js";import{o as S}from"./chunk-XP3TNE3R.js";import{ba as p}from"./chunk-QRHYK2TW.js";import{H as I,y as i}from"./chunk-MOTAOJVG.js";import{f as c,ra as E}from"./chunk-CYTB2B6Q.js";import{b as F}from"./chunk-OL67KS7C.js";import{f as s,o as a,q as n}from"./chunk-SHG7TIBL.js";a();n();var o=s(F()),d=s(S());E();I();var A=({from:f,method:m,chainIndex:h,handleCancel:u,to:g,dappData:k,dappInfo:_,isRpcChain:C},L=p)=>{let[M,e]=(0,o.useState)(!1),[w,r]=(0,o.useState)(""),x=l({handleCancel:u});return(0,d.useMount)(async()=>{try{e(!0),r("");let t=(await L({from:f,method:m,chainIndex:h,to:g,isRpcChain:C,dappData:k,dappInfo:_}))?.popupInfoList;t?.length&&await x(t).catch(c),e(!1)}catch{r(i("wallet_extension_transaction_error_general_check_network"))}finally{e(!1)}}),{addressCheckLoading:M,addressCheckError:w}},z=A;export{z as a};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-JRNVHROF.js.map
