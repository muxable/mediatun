import MediaTunnel from './api';

const mediaTunnel = new MediaTunnel('ws://34.145.147.32:7000/ws');

function publish() {
  navigator.mediaDevices
    .getUserMedia({ video: true, audio: true })
    .then(function (stream) {
      return mediaTunnel.publish(stream);
    })
    .then(function (id) {
      const publishId = document.getElementById('publish-id');
      if (!publishId) {
        return;
      }
      publishId.innerText = id;
    });
}

async function play() {
  const playId = document.getElementById('play-id');
  const video = document.getElementById('video');
  if (
    !(playId instanceof HTMLInputElement) ||
    !(video instanceof HTMLVideoElement)
  ) {
    return;
  }
  var id = playId.value;
  (await mediaTunnel.play(id))(video);
}

const publishEl = document.getElementById('publish');
const playEl = document.getElementById('play');
const videoEl = document.getElementById('video');

if (publishEl) {
  publishEl.addEventListener('click', publish);
}
if (playEl) {
  playEl.addEventListener('click', play);
}
if (videoEl) {
  const id = new URLSearchParams(window.location.search).get('id');
  if (id) {
    var added = false;
    document.addEventListener('click', () => {
      if (added) {
        return;
      }
      mediaTunnel
        .play(id)
        .then((attach) => attach(videoEl as HTMLVideoElement));
      added = true;
    });
  }
}
