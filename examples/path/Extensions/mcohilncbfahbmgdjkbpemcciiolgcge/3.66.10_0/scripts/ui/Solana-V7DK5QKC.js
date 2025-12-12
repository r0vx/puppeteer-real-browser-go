import{c as d}from"./chunk-PMRTKWFS.js";import{G as S,ca as a,m as w}from"./chunk-G44PPNNQ.js";import"./chunk-NCGYY6X6.js";import{b as f}from"./chunk-OL67KS7C.js";import"./chunk-TVBFJITU.js";import"./chunk-32ZJ4F2J.js";import{f as u,o as t,q as n}from"./chunk-SHG7TIBL.js";t();n();var C=u(f());t();n();var h=u(f());var y=()=>{let{useCoin:r}=a.hooks,{accountStore:{computedAccountId:o},walletContractStore:{transactionPayload:s},swapStore:{setSolanaSwapParams:e,sendSolanaTransaction:m,solanaSwapParams:i}}=d(),P=r(501),{coinId:p}=P||{};return(0,h.useMemo)(()=>{try{let c=s?.map(l=>l.payload.transaction),g=c.length>1;return{coinId:p,showDappInfo:!1,showSwitchNetwork:!1,walletId:o,method:"signAllTransactions",params:{message:c},source:"dex",onConfirm:async l=>{let[I]=await S(m({signedTransactions:l,txArray:s,enableJito:g,swapParams:i,walletId:o}));I||e(null)},onCancel:()=>{e(null),a.history?.goBack()}}}catch{return null}},[o,m,e,i,s,p])};var k=()=>{let{SolanaEntry:r}=a.components,o=y();return C.default.createElement(r,{...o})},N=w(k);export{N as default};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=Solana-V7DK5QKC.js.map
