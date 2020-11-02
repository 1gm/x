// ==UserScript==
// @name         Utilities
// @version      0.1
// @description  try to take over the world!
// @author       You
// @match        *
// @grant        none
// @run-at       document-start
// ==/UserScript==

document.addEventListener('DOMContentLoaded', _ => {
    const successBorder = '8px solid #00ffbf';
    const skipBorder = '8px solid yellow';
    const failBorder = '8px solid red';

    window.util = {}
    window.util.download = function(src, el) {
        const url = `http://localhost:8080/download?from=${src}`
        fetch(url).then(res => {
            if (el) {
                if (res.status === 204) {
                    el.style.border = skipBorder;
                } else {
                    el.style.border = successBorder;
                }
            }
            console.log('download success');
        }).catch(err => {
            if (el) {
                el.style.border = failBorder;
            }
            console.log('download fail');
            console.error(err);
        });
    }
}, false);