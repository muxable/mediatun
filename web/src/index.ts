import MediaTunnel from '@muxable/mtun';

var uri = 'wss://' + window.location.host + '/ws';

export function publish() {
  navigator.mediaDevices
    .getUserMedia({ video: true, audio: true })
    .then(function (stream) {
      return new MediaTunnel(uri).publish(stream);
    })
    .then(function (id) {
      const publishId = document.getElementById('publish-id');
      if (!publishId) {
        return;
      }
      publishId.innerText = id;
    });
}

export function play() {
  const playId = document.getElementById('play-id');
  const video = document.getElementById('video');
  if (
    !(playId instanceof HTMLInputElement) ||
    !(video instanceof HTMLVideoElement)
  ) {
    return;
  }
  var id = playId.value;
  new MediaTunnel(uri).play(id).then(function (stream) {
    MediaTunnel.attach(video, stream);
  });
}
