import { Client, LocalStream } from "ion-sdk-js";
import { IonSFUJSONRPCSignal } from "ion-sdk-js/lib/signal/json-rpc-impl";
import { v4 as uuidv4 } from "uuid";

export default class MediaTunnel {
  constructor(private uri: string = "ws://sfu.mediatun.muxable.com:7000/ws") {}

  async publish(stream: MediaStream) {
    const id = uuidv4();

    const signal = new IonSFUJSONRPCSignal(this.uri);

    const client = new Client(signal);

    await new Promise<void>((resolve) => (signal.onopen = resolve));
    await client.join(id, id);

    client.publish(
      new LocalStream(stream, {
        codec: "vp8",
        resolution: "hd",
        audio: false,
        video: true,
        simulcast: false,
      })
    );

    return id;
  }

  async play(id: string) {
    const signal = new IonSFUJSONRPCSignal(this.uri);

    const client = new Client(signal);

    await new Promise<void>((resolve) => (signal.onopen = resolve));
    await client.join(id, id);

    return await new Promise<MediaStream>(
      (resolve) => (client.ontrack = (track, stream) => resolve(stream))
    );
  }

  static async attach(video: HTMLVideoElement, stream: MediaStream) {
    video.srcObject = stream;
    video.autoplay = true;
    video.playsInline = true;
    video.controls = false;
    await video.play();
  }
}
