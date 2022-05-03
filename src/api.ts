export default class MediaTunnel {
  constructor(private uri: string = 'wss://mtun.io/ws') {}

  async play(id: string) {
    let videoStream: MediaStream | null = null;
    let videoElement: HTMLVideoElement | null = null;

    const connect = async () => {
      const signal = new WebSocket(this.uri + '?sid=' + id);

      await new Promise((resolve) => (signal.onopen = resolve));

      const pc = new RTCPeerConnection({
        iceServers: [
          {
            urls: 'stun:stun.l.google.com:19302',
          },
        ],
      });

      pc.addEventListener('icecandidate', (event) => {
        if (!event.candidate) {
          return;
        }
        signal.send(JSON.stringify({ candidate: event.candidate.toJSON() }));
      });

      pc.addEventListener('track', (event) => {
        const stream = event.streams[0];
        if (event.track.kind === 'video') {
          videoStream = stream;
          if (videoElement) {
            videoElement.srcObject = stream;
            videoElement.play();
          }
        } else {
          const audioContext = new AudioContext();
          const sourceNode = audioContext.createMediaStreamSource(stream);
          sourceNode.connect(audioContext.destination);
          // https://stackoverflow.com/a/63844077/86433
          new Audio().srcObject = stream;
        }
      });

      signal.addEventListener('message', async (event) => {
        const data = JSON.parse(event.data);
        if (data.sdp) {
          await pc.setRemoteDescription(new RTCSessionDescription(data.sdp));
          const answer = await pc.createAnswer();
          await pc.setLocalDescription(answer);
          signal.send(JSON.stringify({ sdp: pc.localDescription }));
        }
        if (data.candidate) {
          pc.addIceCandidate(new RTCIceCandidate(data.candidate));
        }
        if (data.error) {
          throw new Error(data.error);
        }
      });

      signal.addEventListener('close', () => connect());
    };

    await connect();

    return (video: HTMLVideoElement) => {
      videoElement = video;
      if (videoStream) {
        video.srcObject = videoStream;
        video.play();
      }
    };
  }
}
