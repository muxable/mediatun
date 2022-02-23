import { c as createCommonjsModule } from './common/_commonjsHelpers-668e6127.js';

var stream = createCommonjsModule(function (module, exports) {
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeRemote = exports.LocalStream = exports.VideoConstraints = void 0;
const resolutions = ['qvga', 'vga', 'shd', 'hd', 'fhd', 'qhd'];
exports.VideoConstraints = {
    qvga: {
        resolution: {
            width: { ideal: 320 },
            height: { ideal: 180 },
            frameRate: {
                ideal: 15,
                max: 30,
            },
        },
        encodings: {
            maxBitrate: 150000,
            maxFramerate: 15.0,
        },
    },
    vga: {
        resolution: {
            width: { ideal: 640 },
            height: { ideal: 360 },
            frameRate: {
                ideal: 30,
                max: 60,
            },
        },
        encodings: {
            maxBitrate: 500000,
            maxFramerate: 30.0,
        },
    },
    shd: {
        resolution: {
            width: { ideal: 960 },
            height: { ideal: 540 },
            frameRate: {
                ideal: 30,
                max: 60,
            },
        },
        encodings: {
            maxBitrate: 1200000,
            maxFramerate: 30.0,
        },
    },
    hd: {
        resolution: {
            width: { ideal: 1280 },
            height: { ideal: 720 },
            frameRate: {
                ideal: 30,
                max: 60,
            },
        },
        encodings: {
            maxBitrate: 2500000,
            maxFramerate: 30.0,
        },
    },
    fhd: {
        resolution: {
            width: { ideal: 1920 },
            height: { ideal: 1080 },
            frameRate: {
                ideal: 30,
                max: 60,
            },
        },
        encodings: {
            maxBitrate: 4000000,
            maxFramerate: 30.0,
        },
    },
    qhd: {
        resolution: {
            width: { ideal: 2560 },
            height: { ideal: 1440 },
            frameRate: {
                ideal: 30,
                max: 60,
            },
        },
        encodings: {
            maxBitrate: 8000000,
            maxFramerate: 30.0,
        },
    },
};
const defaults = {
    resolution: 'hd',
    codec: 'vp8',
    audio: true,
    video: true,
    simulcast: false,
};
class LocalStream extends MediaStream {
    constructor(stream, constraints) {
        super(stream);
        this.constraints = constraints;
    }
    static async getUserMedia(constraints = defaults) {
        const stream = await navigator.mediaDevices.getUserMedia({
            audio: LocalStream.computeAudioConstraints({
                ...defaults,
                ...constraints,
            }),
            video: LocalStream.computeVideoConstraints({
                ...defaults,
                ...constraints,
            }),
        });
        return new LocalStream(stream, {
            ...defaults,
            ...constraints,
        });
    }
    static async getDisplayMedia(constraints = {
        codec: 'vp8',
        resolution: 'hd',
        audio: false,
        video: true,
        simulcast: false,
    }) {
        // @ts-ignore
        const stream = await navigator.mediaDevices.getDisplayMedia(constraints);
        return new LocalStream(stream, {
            ...defaults,
            ...constraints,
        });
    }
    static computeAudioConstraints(constraints) {
        return constraints.audio;
    }
    static computeVideoConstraints(constraints) {
        if (constraints.video instanceof Object) {
            return constraints.video;
        }
        else if (constraints.video && constraints.resolution) {
            return {
                ...exports.VideoConstraints[constraints.resolution].resolution,
            };
        }
        return constraints.video;
    }
    getTrack(kind) {
        let tracks;
        if (kind === 'video') {
            tracks = this.getVideoTracks();
            return tracks.length > 0 ? this.getVideoTracks()[0] : undefined;
        }
        tracks = this.getAudioTracks();
        return tracks.length > 0 ? this.getAudioTracks()[0] : undefined;
    }
    async getNewTrack(kind) {
        const stream = await navigator.mediaDevices.getUserMedia({
            [kind]: kind === 'video'
                ? LocalStream.computeVideoConstraints(this.constraints)
                : LocalStream.computeAudioConstraints(this.constraints),
        });
        return stream.getTracks()[0];
    }
    publishTrack(track) {
        if (this.pc) {
            const init = {
                streams: [this],
                direction: 'sendonly',
            };
            if (track.kind === 'video') {
                if (this.encodingParams) {
                    init.sendEncodings = this.encodingParams;
                }
                else if (this.constraints.simulcast) {
                    const idx = resolutions.indexOf(this.constraints.resolution);
                    const encodings = [
                        {
                            rid: 'f',
                            maxBitrate: exports.VideoConstraints[resolutions[idx]].encodings.maxBitrate,
                            maxFramerate: exports.VideoConstraints[resolutions[idx]].encodings.maxFramerate,
                        },
                    ];
                    if (idx - 1 >= 0) {
                        encodings.push({
                            rid: 'h',
                            scaleResolutionDownBy: 2.0,
                            maxBitrate: exports.VideoConstraints[resolutions[idx - 1]].encodings.maxBitrate,
                            maxFramerate: exports.VideoConstraints[resolutions[idx - 1]].encodings.maxFramerate,
                        });
                    }
                    if (idx - 2 >= 0) {
                        encodings.push({
                            rid: 'q',
                            scaleResolutionDownBy: 4.0,
                            maxBitrate: exports.VideoConstraints[resolutions[idx - 2]].encodings.maxBitrate,
                            maxFramerate: exports.VideoConstraints[resolutions[idx - 2]].encodings.maxFramerate,
                        });
                    }
                    init.sendEncodings = encodings;
                }
                else {
                    init.sendEncodings = [exports.VideoConstraints[this.constraints.resolution].encodings];
                }
            }
            const transceiver = this.pc.addTransceiver(track, init);
            this.setPreferredCodec(transceiver, track.kind);
        }
    }
    setPreferredCodec(transceiver, kind) {
        if ('setCodecPreferences' in transceiver) {
            const cap = RTCRtpSender.getCapabilities(kind);
            if (!cap)
                return;
            let selCodec;
            // 42e01f for safari/chrome/firefox cross-browser compatibility
            if (kind === 'video' && this.constraints.codec && this.constraints.codec.toLowerCase() === 'h264') {
                this.constraints.preferredCodecProfile = '42e01f';
            }
            if (this.constraints.preferredCodecProfile && kind === 'video') {
                const allCodecProfiles = cap.codecs.filter((c) => c.mimeType.toLowerCase() === `video/${this.constraints.codec.toLowerCase()}`);
                if (!allCodecProfiles) {
                    return;
                }
                selCodec = allCodecProfiles.find((c) => { var _a; return c.sdpFmtpLine && ((_a = c.sdpFmtpLine) === null || _a === void 0 ? void 0 : _a.indexOf(`profile-level-id=${this.constraints.preferredCodecProfile}`)) >= 0; });
                if (!selCodec) {
                    // get first one
                    selCodec = allCodecProfiles[0];
                }
            }
            else {
                selCodec = cap.codecs.find((c) => c.mimeType.toLowerCase() === `video/${this.constraints.codec.toLowerCase()}` ||
                    c.mimeType.toLowerCase() === `audio/opus`);
            }
            if (selCodec) {
                transceiver.setCodecPreferences([selCodec]);
            }
        }
    }
    updateTrack(next, prev) {
        this.addTrack(next);
        // If published, replace published track with track from new device
        if (prev) {
            this.removeTrack(prev);
            prev.stop();
            if (this.pc) {
                this.pc.getSenders().forEach(async (sender) => {
                    var _a, _b;
                    if (((_a = sender === null || sender === void 0 ? void 0 : sender.track) === null || _a === void 0 ? void 0 : _a.kind) === next.kind) {
                        (_b = sender.track) === null || _b === void 0 ? void 0 : _b.stop();
                        sender.replaceTrack(next);
                    }
                });
            }
        }
        else {
            this.addTrack(next);
            if (this.pc) {
                this.publishTrack(next);
            }
        }
    }
    initAudioEmptyTrack() {
        // @ts-ignore
        const AudioContext = window.AudioContext || window.webkitAudioContext;
        const ctx = new AudioContext();
        const oscillator = ctx.createOscillator();
        oscillator.frequency.setValueAtTime(20000, ctx.currentTime);
        const dst = oscillator.connect(ctx.createMediaStreamDestination());
        oscillator.start();
        return dst.stream.getAudioTracks()[0];
    }
    initVideoEmptyTrack(width, height) {
        var _a;
        const canvas = Object.assign(document.createElement('canvas'), { width, height });
        (_a = canvas.getContext('2d')) === null || _a === void 0 ? void 0 : _a.fillRect(0, 0, width, height);
        const stream = canvas.captureStream();
        return stream.getVideoTracks()[0];
    }
    publish(transport, encodingParams) {
        this.pc = transport.pc;
        this.api = transport.api;
        this.encodingParams = encodingParams;
        this.getTracks().forEach(this.publishTrack.bind(this));
    }
    unpublish() {
        if (this.pc) {
            const tracks = this.getTracks();
            this.pc.getSenders().forEach((s) => {
                if (s.track && tracks.includes(s.track)) {
                    this.pc.removeTrack(s);
                }
            });
        }
    }
    async switchDevice(kind, deviceId) {
        this.constraints = {
            ...this.constraints,
            [kind]: this.constraints[kind] instanceof Object
                ? {
                    ...this.constraints[kind],
                    deviceId,
                }
                : { deviceId },
        };
        const prev = this.getTrack(kind);
        const next = await this.getNewTrack(kind);
        this.updateTrack(next, prev);
    }
    mute(kind) {
        const track = this.getTrack(kind);
        if (track && this.constraints.sendEmptyOnMute) {
            const emptyTrack = kind === 'audio'
                ? this.initAudioEmptyTrack()
                : this.initVideoEmptyTrack((track === null || track === void 0 ? void 0 : track.getSettings().width) || 640, (track === null || track === void 0 ? void 0 : track.getSettings().height) || 360);
            emptyTrack.enabled = false;
            this.updateTrack(emptyTrack, track);
            return;
        }
        if (track) {
            track.stop();
        }
    }
    async unmute(kind) {
        const prev = this.getTrack(kind);
        const track = await this.getNewTrack(kind);
        this.updateTrack(track, prev);
    }
    async enableLayers(layers) {
        const call = {
            streamId: this.id,
            layers,
        };
        const callStr = JSON.stringify(call);
        if (this.api) {
            if (this.api.readyState !== 'open') {
                // queue call if we aren't open yet
                this.api.onopen = () => { var _a; return (_a = this.api) === null || _a === void 0 ? void 0 : _a.send(JSON.stringify(call)); };
            }
            else {
                this.api.send(JSON.stringify(call));
            }
        }
        const layerValues = ['high', 'medium', 'low'];
        await Promise.all(layerValues.map(async (layer) => {
            await this.updateMediaEncodingParams({ active: layers.includes(layer) }, layer);
        }));
    }
    async updateMediaEncodingParams(encodingParams, layer) {
        var _a;
        if (!this.pc)
            return;
        const tracks = this.getTracks();
        await Promise.all((_a = this.pc) === null || _a === void 0 ? void 0 : _a.getSenders().filter((sender) => sender.track && tracks.includes(sender.track)).map(async (sender) => {
            const params = sender.getParameters();
            if (!params.encodings) {
                params.encodings = [{}];
            }
            let idx = 0;
            if (this.constraints.simulcast && layer) {
                const rid = layer === 'high' ? 'f' : layer === 'medium' ? 'h' : 'q';
                idx = params.encodings.findIndex((encoding) => encoding.rid === rid);
                if (params.encodings.length < idx + 1)
                    return;
            }
            params.encodings[idx] = {
                ...params.encodings[idx],
                ...encodingParams,
            };
            await sender.setParameters(params);
        }));
    }
}
exports.LocalStream = LocalStream;
function makeRemote(stream, transport) {
    const remote = stream;
    remote.audio = true;
    remote.video = 'none';
    remote.framerate = 'high';
    remote._videoPreMute = 'high';
    const select = () => {
        const call = {
            streamId: remote.id,
            video: remote.video,
            audio: remote.audio,
            framerate: remote.framerate,
        };
        if (transport.api) {
            if (transport.api.readyState !== 'open') {
                // queue call if we aren't open yet
                transport.api.onopen = () => { var _a; return (_a = transport.api) === null || _a === void 0 ? void 0 : _a.send(JSON.stringify(call)); };
            }
            else {
                transport.api.send(JSON.stringify(call));
            }
        }
    };
    remote.preferLayer = (layer) => {
        remote.video = layer;
        select();
    };
    remote.preferFramerate = (layer) => {
        remote.framerate = layer;
        select();
    };
    remote.mute = (kind) => {
        if (kind === 'audio') {
            remote.audio = false;
        }
        else if (kind === 'video') {
            remote._videoPreMute = remote.video;
            remote.video = 'none';
        }
        select();
    };
    remote.unmute = (kind) => {
        if (kind === 'audio') {
            remote.audio = true;
        }
        else if (kind === 'video') {
            remote.video = remote._videoPreMute;
        }
        select();
    };
    return remote;
}
exports.makeRemote = makeRemote;
});

var client = createCommonjsModule(function (module, exports) {
Object.defineProperty(exports, "__esModule", { value: true });
exports.Transport = void 0;

const API_CHANNEL = 'ion-sfu';
const ERR_NO_SESSION = 'no active session, join first';
var Role;
(function (Role) {
    Role[Role["pub"] = 0] = "pub";
    Role[Role["sub"] = 1] = "sub";
})(Role || (Role = {}));
class Transport {
    constructor(role, signal, config) {
        this.signal = signal;
        this.pc = new RTCPeerConnection(config);
        this.candidates = [];
        if (role === Role.pub) {
            this.pc.createDataChannel(API_CHANNEL);
        }
        this.pc.onicecandidate = ({ candidate }) => {
            if (candidate) {
                this.signal.trickle({ target: role, candidate });
            }
        };
        this.pc.oniceconnectionstatechange = async (e) => {
            if (this.pc.iceConnectionState === 'disconnected') {
                if (this.pc.restartIce !== undefined) {
                    // this will trigger onNegotiationNeeded
                    this.pc.restartIce();
                }
            }
        };
    }
}
exports.Transport = Transport;
class Client {
    constructor(signal, config = {
        codec: 'vp8',
        iceServers: [
            {
                urls: ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302'],
            },
        ],
    }) {
        this.signal = signal;
        this.config = config;
        signal.onnegotiate = this.negotiate.bind(this);
        signal.ontrickle = this.trickle.bind(this);
    }
    async join(sid, uid) {
        this.transports = {
            [Role.pub]: new Transport(Role.pub, this.signal, this.config),
            [Role.sub]: new Transport(Role.sub, this.signal, this.config),
        };
        this.transports[Role.sub].pc.ontrack = (ev) => {
            const stream$1 = ev.streams[0];
            const remote = stream.makeRemote(stream$1, this.transports[Role.sub]);
            if (this.ontrack) {
                this.ontrack(ev.track, remote);
            }
        };
        const apiReady = new Promise((resolve) => {
            this.transports[Role.sub].pc.ondatachannel = (ev) => {
                if (ev.channel.label === API_CHANNEL) {
                    this.transports[Role.sub].api = ev.channel;
                    this.transports[Role.pub].api = ev.channel;
                    ev.channel.onmessage = (e) => {
                        try {
                            const msg = JSON.parse(e.data);
                            this.processChannelMessage(msg);
                        }
                        catch (err) {
                            /* tslint:disable-next-line:no-console */
                            console.error(err);
                        }
                    };
                    resolve();
                    return;
                }
                if (this.ondatachannel) {
                    this.ondatachannel(ev);
                }
            };
        });
        const offer = await this.transports[Role.pub].pc.createOffer();
        await this.transports[Role.pub].pc.setLocalDescription(offer);
        const answer = await this.signal.join(sid, uid, offer);
        await this.transports[Role.pub].pc.setRemoteDescription(answer);
        this.transports[Role.pub].candidates.forEach((c) => this.transports[Role.pub].pc.addIceCandidate(c));
        this.transports[Role.pub].pc.onnegotiationneeded = this.onNegotiationNeeded.bind(this);
        return apiReady;
    }
    leave() {
        if (this.transports) {
            Object.values(this.transports).forEach((t) => t.pc.close());
            delete this.transports;
        }
    }
    getPubStats(selector) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        return this.transports[Role.pub].pc.getStats(selector);
    }
    getSubStats(selector) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        return this.transports[Role.sub].pc.getStats(selector);
    }
    publish(stream, encodingParams) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        stream.publish(this.transports[Role.pub], encodingParams);
    }
    restartIce() {
        this.renegotiate(true);
    }
    createDataChannel(label) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        return this.transports[Role.pub].pc.createDataChannel(label);
    }
    close() {
        if (this.transports) {
            Object.values(this.transports).forEach((t) => t.pc.close());
        }
        this.signal.close();
    }
    trickle({ candidate, target }) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        if (this.transports[target].pc.remoteDescription) {
            this.transports[target].pc.addIceCandidate(candidate);
        }
        else {
            this.transports[target].candidates.push(candidate);
        }
    }
    async negotiate(description) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        let answer;
        try {
            await this.transports[Role.sub].pc.setRemoteDescription(description);
            this.transports[Role.sub].candidates.forEach((c) => this.transports[Role.sub].pc.addIceCandidate(c));
            this.transports[Role.sub].candidates = [];
            answer = await this.transports[Role.sub].pc.createAnswer();
            await this.transports[Role.sub].pc.setLocalDescription(answer);
            this.signal.answer(answer);
        }
        catch (err) {
            /* tslint:disable-next-line:no-console */
            console.error(err);
            if (this.onerrnegotiate)
                this.onerrnegotiate(Role.sub, err, description, answer);
        }
    }
    onNegotiationNeeded() {
        this.renegotiate(false);
    }
    async renegotiate(iceRestart) {
        if (!this.transports) {
            throw Error(ERR_NO_SESSION);
        }
        let offer;
        let answer;
        try {
            offer = await this.transports[Role.pub].pc.createOffer({ iceRestart });
            await this.transports[Role.pub].pc.setLocalDescription(offer);
            answer = await this.signal.offer(offer);
            await this.transports[Role.pub].pc.setRemoteDescription(answer);
        }
        catch (err) {
            /* tslint:disable-next-line:no-console */
            console.error(err);
            if (this.onerrnegotiate)
                this.onerrnegotiate(Role.pub, err, offer, answer);
        }
    }
    processChannelMessage(msg) {
        if (msg.method !== undefined && msg.params !== undefined) {
            switch (msg.method) {
                case 'audioLevels':
                    if (this.onspeaker) {
                        this.onspeaker(msg.params);
                    }
                    break;
                case 'activeLayer':
                    if (this.onactivelayer) {
                        this.onactivelayer(msg.params);
                    }
                    break;
                // do nothing
            }
        }
        else {
            // legacy channel message - payload contains audio levels
            if (this.onspeaker) {
                this.onspeaker(msg);
            }
        }
    }
}
exports.default = Client;
});

var lib = createCommonjsModule(function (module, exports) {
Object.defineProperty(exports, "__esModule", { value: true });
exports.LocalStream = exports.Client = void 0;

exports.Client = client.default;

Object.defineProperty(exports, "LocalStream", { enumerable: true, get: function () { return stream.LocalStream; } });
});

var Client = lib.Client;
var LocalStream = lib.LocalStream;
export { Client, LocalStream };
