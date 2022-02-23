import {Client, LocalStream} from "../_snowpack/pkg/ion-sdk-js.js";
import {IonSFUJSONRPCSignal} from "../_snowpack/pkg/ion-sdk-js/lib/signal/json-rpc-impl.js";
import {v4 as uuidv4} from "../_snowpack/pkg/uuid.js";
export default class MediaTunnel {
  constructor(uri = "wss://mtun.io/ws") {
    this.uri = uri;
  }
  async publish(stream) {
    const id = uuidv4();
    const signal = new IonSFUJSONRPCSignal(this.uri);
    const client = new Client(signal);
    await new Promise((resolve) => signal.onopen = resolve);
    await client.join(id, id);
    client.publish(new LocalStream(stream, {
      codec: "vp8",
      resolution: "hd",
      audio: false,
      video: true,
      simulcast: false
    }));
    return id;
  }
  async play(id) {
    const signal = new IonSFUJSONRPCSignal(this.uri);
    const client = new Client(signal);
    await new Promise((resolve) => signal.onopen = resolve);
    let videoStream = null;
    let videoElement = null;
    client.ontrack = (track, stream) => {
      console.log("track", track, stream);
      if (track.kind === "video") {
        videoStream = stream;
        if (videoElement) {
          videoElement.srcObject = stream;
          videoElement.play();
        }
      } else {
        const audioContext = new AudioContext();
        const sourceNode = audioContext.createMediaStreamSource(stream);
        sourceNode.connect(audioContext.destination);
        new Audio().srcObject = stream;
      }
    };
    await client.join(id, uuidv4());
    return (video) => {
      videoElement = video;
      if (videoStream) {
        video.srcObject = videoStream;
        video.play();
      }
    };
  }
}
