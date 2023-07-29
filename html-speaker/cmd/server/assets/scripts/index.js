window.$ = document.querySelector.bind(document);
window.$$ = document.querySelectorAll.bind(document);

document.addEventListener('DOMContentLoaded', function () {
    const socket = new ReconnectingWebSocket((location.protocol == 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/ws');
    socket.addEventListener('message', function (event) {
        console.log(event.data)
        appendMessage(`${event.data}`)
    });
});

const appendMessage = (function () {
    const el = $("#messages");
    return function (data) {
        el?.insertAdjacentHTML('beforeend',`<li class="list-group-item p-1"><small>${data}</small></li>`);
    };
})();