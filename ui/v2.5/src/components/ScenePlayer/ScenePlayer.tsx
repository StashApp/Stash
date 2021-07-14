/* eslint-disable @typescript-eslint/no-explicit-any */
import React from "react";
import ReactJWPlayer from "react-jw-player";
import * as GQL from "src/core/generated-graphql";
import { useConfiguration } from "src/core/StashService";
import { JWUtils } from "src/utils";
import { ScenePlayerScrubber } from "./ScenePlayerScrubber";
import { Interactive } from "../../utils/interactive";

interface IScenePlayerProps {
  className?: string;
  scene: GQL.SceneDataFragment;
  sceneStreams: GQL.SceneStreamEndpoint[];
  timestamp: number;
  autoplay?: boolean;
  onReady?: () => void;
  onSeeked?: () => void;
  onTime?: () => void;
  onComplete?: () => void;
  config?: GQL.ConfigInterfaceDataFragment;
}
interface IScenePlayerState {
  scrubberPosition: number;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config: Record<string, any>;
  interactiveClient: Interactive;
}
export class ScenePlayerImpl extends React.Component<
  IScenePlayerProps,
  IScenePlayerState
> {
  private static isDirectStream(src?: string) {
    if (!src) {
      return false;
    }

    const url = new URL(src);
    return url.pathname.endsWith("/stream");
  }

  // Typings for jwplayer are, unfortunately, very lacking
  private player: any;
  private playlist: any;
  private lastTime = 0;

  constructor(props: IScenePlayerProps) {
    super(props);
    this.onReady = this.onReady.bind(this);
    this.onSeeked = this.onSeeked.bind(this);
    this.onTime = this.onTime.bind(this);

    this.onScrubberSeek = this.onScrubberSeek.bind(this);
    this.onScrubberScrolled = this.onScrubberScrolled.bind(this);
    this.state = {
      scrubberPosition: 0,
      config: this.makeJWPlayerConfig(props.scene),
      interactiveClient: new Interactive(
        this.props.config?.handyKey || "",
        this.props.config?.scriptOffset || 0
      ),
    };

    // Default back to Direct Streaming
    localStorage.removeItem("jwplayer.qualityLabel");
  }
  public UNSAFE_componentWillReceiveProps(props: IScenePlayerProps) {
    if (props.scene !== this.props.scene) {
      this.setState((state) => ({
        ...state,
        config: this.makeJWPlayerConfig(this.props.scene),
      }));
    }
  }

  public componentDidUpdate(prevProps: IScenePlayerProps) {
    if (prevProps.timestamp !== this.props.timestamp) {
      this.player.seek(this.props.timestamp);
    }
  }

  onIncrease() {
    const currentPlaybackRate = this.player ? this.player.getPlaybackRate() : 1;
    this.player.setPlaybackRate(currentPlaybackRate + 0.5);
  }
  onDecrease() {
    const currentPlaybackRate = this.player ? this.player.getPlaybackRate() : 1;
    this.player.setPlaybackRate(currentPlaybackRate - 0.5);
  }

  onReset() {
    this.player.setPlaybackRate(1);
  }
  onPause() {
    if (this.player.getState().paused) {
      this.player.play();
    } else {
      this.player.pause();
    }
  }

  private onReady() {
    this.player = JWUtils.getPlayer();

    this.player.on("error", (err: any) => {
      if (err && err.code === 224003) {
        // When jwplayer has been requested to play but the browser doesn't support the video format.
        this.handleError(true);
      }
    });

    //
    this.player.on("meta", (metadata: any) => {
      if (
        metadata.metadataType === "media" &&
        !metadata.width &&
        !metadata.height
      ) {
        // Occurs during preload when videos with supported audio/unsupported video are preloaded.
        // Treat this as a decoding error and try the next source without playing.
        // However on Safari we get an media event when m3u8 is loaded which needs to be ignored.
        const currentFile = this.player.getPlaylistItem().file;
        if (currentFile != null && !currentFile.includes("m3u8")) {
          const state = this.player.getState();
          const play = state === "buffering" || state === "playing";
          this.handleError(play);
        }
      }
    });

    this.player.on("firstFrame", () => {
      if (this.props.timestamp > 0) {
        this.player.seek(this.props.timestamp);
      }
    });

    this.player.on("play", () => {
      if (this.props.scene.interactive) {
        this.state.interactiveClient.play(this.player.getPosition());
      }
    });

    this.player.on("pause", () => {
      if (this.props.scene.interactive) {
        this.state.interactiveClient.pause();
      }
    });

    if (this.props.scene.interactive) {
      this.state.interactiveClient.uploadScript(
        this.props.scene.paths.funscript || ""
      );
    }
  }

  private onSeeked() {
    const position = this.player.getPosition();
    this.setState({ scrubberPosition: position });
    this.player.play();
  }

  private onTime() {
    const position = this.player.getPosition();
    const difference = Math.abs(position - this.lastTime);
    if (difference > 1) {
      this.lastTime = position;
      this.setState({ scrubberPosition: position });
      if (this.props.scene.interactive) {
        this.state.interactiveClient.ensurePlaying(position);
      }
    }
  }

  private onComplete() {
    if (this.props?.onComplete) {
      this.props.onComplete();
    }
  }

  private onScrubberSeek(seconds: number) {
    this.player.seek(seconds);
  }

  private onScrubberScrolled() {
    this.player.pause();
  }

  private handleError(play: boolean) {
    const currentFile = this.player.getPlaylistItem();
    if (currentFile) {
      // eslint-disable-next-line no-console
      console.log(`Source failed: ${currentFile.file}`);
    }

    if (this.tryNextStream()) {
      // eslint-disable-next-line no-console
      console.log(
        `Trying next source in playlist: ${this.playlist.sources[0].file}`
      );
      this.player.load(this.playlist);
      if (play) {
        this.player.play();
      }
    }
  }

  private shouldRepeat(scene: GQL.SceneDataFragment) {
    const maxLoopDuration = this.props?.config?.maximumLoopDuration ?? 0;
    return (
      !!scene.file.duration &&
      !!maxLoopDuration &&
      scene.file.duration < maxLoopDuration
    );
  }

  private tryNextStream() {
    if (this.playlist.sources.length > 1) {
      this.playlist.sources.shift();
      return true;
    }

    return false;
  }

  private makePlaylist() {
    const { scene } = this.props;

    return {
      image: scene.paths.screenshot,
      tracks: [
        {
          file: scene.paths.vtt,
          kind: "thumbnails",
        },
        {
          file: scene.paths.chapters_vtt,
          kind: "chapters",
        },
      ],
      sources: this.props.sceneStreams.map((s) => {
        return {
          file: s.url,
          type: s.mime_type,
          label: s.label,
        };
      }),
    };
  }

  private makeJWPlayerConfig(scene: GQL.SceneDataFragment) {
    if (!scene.paths.stream) {
      return {};
    }

    const repeat = this.shouldRepeat(scene);
    const getDurationHook = () => {
      return this.props.scene.file.duration ?? null;
    };

    const seekHook = (seekToPosition: number, _videoTag: HTMLVideoElement) => {
      if (!_videoTag.src || _videoTag.src.endsWith(".m3u8")) {
        return false;
      }

      if (ScenePlayerImpl.isDirectStream(_videoTag.src)) {
        if (_videoTag.dataset.start) {
          /* eslint-disable-next-line no-param-reassign */
          _videoTag.dataset.start = "0";
        }

        // direct stream - fall back to default
        return false;
      }

      // remove the start parameter
      const srcUrl = new URL(_videoTag.src);
      srcUrl.searchParams.delete("start");

      /* eslint-disable no-param-reassign */
      _videoTag.dataset.start = seekToPosition.toString();
      srcUrl.searchParams.append("start", seekToPosition.toString());
      _videoTag.src = srcUrl.toString();
      /* eslint-enable no-param-reassign */

      _videoTag.play();

      // return true to indicate not to fall through to default
      return true;
    };

    const getCurrentTimeHook = (_videoTag: HTMLVideoElement) => {
      const start = Number.parseFloat(_videoTag.dataset?.start ?? "0");
      return _videoTag.currentTime + start;
    };

    this.playlist = this.makePlaylist();

    const ret = {
      playlist: this.playlist,
      image: scene.paths.screenshot,
      width: "100%",
      height: "100%",
      floating: {
        dismissible: true,
      },
      cast: {},
      primary: "html5",
      autostart:
        this.props.autoplay ||
        (this.props.config ? this.props.config.autostartVideo : false) ||
        this.props.timestamp > 0,
      repeat,
      playbackRateControls: true,
      playbackRates: [0.75, 1, 1.5, 2, 3, 4],
      getDurationHook,
      seekHook,
      getCurrentTimeHook,
    };

    return ret;
  }

  public render() {
    let className =
      this.props.className ?? "w-100 col-sm-9 m-sm-auto no-gutter";
    const sceneFile = this.props.scene.file;

    if (
      sceneFile.height &&
      sceneFile.width &&
      sceneFile.height > sceneFile.width
    ) {
      className += " portrait";
    }

    return (
      <div id="jwplayer-container" className={className}>
        <ReactJWPlayer
          playerId={JWUtils.playerID}
          playerScript="/jwplayer/jwplayer.js"
          customProps={this.state.config}
          onReady={this.onReady}
          onSeeked={this.onSeeked}
          onTime={this.onTime}
          onOneHundredPercent={() => this.onComplete()}
        />
        <ScenePlayerScrubber
          scene={this.props.scene}
          position={this.state.scrubberPosition}
          onSeek={this.onScrubberSeek}
          onScrolled={this.onScrubberScrolled}
        />
      </div>
    );
  }
}

export const ScenePlayer: React.FC<IScenePlayerProps> = (
  props: IScenePlayerProps
) => {
  const config = useConfiguration();

  return (
    <ScenePlayerImpl
      {...props}
      config={
        config.data && config.data.configuration
          ? config.data.configuration.interface
          : undefined
      }
    />
  );
};
