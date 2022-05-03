import MediaTunnel from './api';

const mediaTunnel = new MediaTunnel('ws://localhost:7000/');

const videoEl = document.getElementById('video');

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
