import{b as c}from"./chunk-7C7MBE2C.js";import{g as u}from"./chunk-AZNX5TUC.js";import{o as k}from"./chunk-XP3TNE3R.js";import{Ab as m,Fd as a,Jb as W,Ma as n,Na as C,Pd as _}from"./chunk-QHXUL3YM.js";import{H as b,p as f}from"./chunk-MOTAOJVG.js";import{j as i,ra as L}from"./chunk-CYTB2B6Q.js";import{b as E}from"./chunk-OL67KS7C.js";import{f as r,o,q as s}from"./chunk-SHG7TIBL.js";o();s();var l=r(E()),p=r(k());L();b();_();C();W();var T="update_defi_list",M=()=>{let d=u(),I=a(),t=(0,p.useRequest)(async()=>{let D=await m(n.getDefiList,{accountId:d});return i(D,["data","platformListByAccountId","0","platformListVoList"],[]).filter(g=>I.find(h=>Number(h.netWorkId)===g.chainId))},{manual:!0}),e=c("invest-DeFi",{onError:t.refresh,pollingInterval:30*1e3});return(0,l.useEffect)(()=>{e&&t.refresh()},[e]),f.listen(T,t.refresh,!1),t};export{T as a,M as b};

window.inOKXExtension = true;
window.inMiniApp = false;
window.ASSETS_BUILD_TYPE = "publish";

//# sourceMappingURL=chunk-HUOF3MWN.js.map
