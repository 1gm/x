import ReconnectingWebSocket from './websocket';
import { AudioPlayer } from './audio';

document.addEventListener('DOMContentLoaded', function () {
    const audioPlayer = new AudioPlayer();

    // initialize our websocket connection
    const url = (location.protocol == 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/ws';
    console.log(`websocket url: ${url}`);
    const socket = new ReconnectingWebSocket(url);
    socket.addEventListener('message', function (event) {
        appendMessage(`event received at ${new Date().getTime()}`);
        audioPlayer.queueTrack(event.data);
    });

});

const appendMessage = (function () {
    const el = document.querySelector("#messages");
    return function (data) {
        el?.insertAdjacentHTML('beforeend',`<li class="list-group-item p-1"><small>${data}</small></li>`);
    };
})();