import { Client, LocalStream } from 'ion-sdk-js';
import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
import { v4 as uuidv4 } from 'uuid';

export default class MediaTunnel {
  constructor(private uri: string = 'wss://mtun.io/ws') {}

  async publish(stream: MediaStream) {
    const id = uuidv4();

    const signal = new IonSFUJSONRPCSignal(this.uri);

    const client = new Client(signal);

    await new Promise<void>((resolve) => (signal.onopen = resolve));
    await client.join(id, id);

    client.publish(
      new LocalStream(stream, {
        codec: 'vp8',
        resolution: 'hd',
        audio: false,
        video: true,
        simulcast: false,
      }),
    );

    return id;
  }

  async play(id: string) {
    const signal = new IonSFUJSONRPCSignal(this.uri);

    const client = new Client(signal);

    await new Promise<void>((resolve) => (signal.onopen = resolve));

    let videoStream: MediaStream | null = null;
    let videoElement: HTMLVideoElement | null = null;

    client.ontrack = (track, stream) => {
      console.log('track', track, stream);
      if (track.kind === 'video') {
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
    };

    await client.join(id, uuidv4());

    return (video: HTMLVideoElement) => {
      videoElement = video;
      if (videoStream) {
        video.srcObject = videoStream;
        video.play();
      }
    };
  }
}
