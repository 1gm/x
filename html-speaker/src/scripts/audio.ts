enum AudioPlayerState {
    Idle,
    Playing,
    Waiting
}

export interface AudioPlayerOptions {
    Delay?: number
    AudioElement?: HTMLAudioElement;
}

export class AudioPlayer {
    private readonly audio: HTMLAudioElement;
    private state: AudioPlayerState = AudioPlayerState.Idle;
    private queue: Array<string> = [];
    private history: Array<string> = [];
    private readonly delay: number;

    constructor(opts?: AudioPlayerOptions) {
        this.delay = opts?.Delay ?? 1500;
        this.audio = opts?.AudioElement ?? document.createElement("audio");
        this.audio.addEventListener("ended", this.ended.bind(this));
    }

    // queueTracks accepts base64 encoded audio data
    queueTrack(data: string): void {
        this.queue.push(data);
        if(this.state == AudioPlayerState.Idle) {
            this.state = AudioPlayerState.Waiting;
            setTimeout(() => {
                this.playNextTrack();
            }, this.delay);
        }
    }

    // replay plays the last played track
    replay(): void {
        if(this.history.length > 0) {
            const lastIndex = this.history.length - 1;
            this.queueTrack(this.history[lastIndex]);
        }
    }

    private playNextTrack(): void {
        if(this.state == AudioPlayerState.Waiting || this.state == AudioPlayerState.Idle) {
            const current = this.queue.shift();
            if(typeof current === "string"){
                this.history.push(current);
                this.audio.src = `data:audio/ogg;base64,${current}`;
                this.audio.play();
                this.state = AudioPlayerState.Playing;
            }
        }
    }

    private ended(ev: Event): void {
        this.state = AudioPlayerState.Idle;
        if(this.queue.length > 0) {
            this.state = AudioPlayerState.Waiting;
            setTimeout(() => {
                this.playNextTrack();
            }, this.delay);
        }
    }
}