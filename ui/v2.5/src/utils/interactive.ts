import Handy from "thehandy";

interface IFunscript {
  actions: Array<IAction>;
}

interface IAction {
  at: number;
  pos: number;
}

// Copied from handy-js-sdk under MIT license, with modifications. (It's not published to npm)
// Converting to CSV first instead of uploading Funscripts will reduce uploaded file size.
function convertFunscriptToCSV(funscript: IFunscript) {
  const lineTerminator = "\r\n";
  if (funscript?.actions?.length > 0) {
    return funscript.actions.reduce((prev: string, curr: IAction) => {
      return `${prev}${curr.at},${curr.pos}${lineTerminator}`;
    }, `#Created by stash.app ${new Date().toUTCString()}\n`);
  }
  throw new Error("Not a valid funscript");
}

// Interactive currently uses the Handy API, but could be expanded to use buttplug.io
// via buttplugio/buttplug-rs-ffi's WASM module.
export class Interactive {
  private _connected: boolean;
  private _playing: boolean;
  private _scriptOffset: number;
  private _handy: Handy;

  constructor(handyKey: string, scriptOffset: number) {
    this._handy = new Handy();
    this._handy.connectionKey = handyKey;
    this._scriptOffset = scriptOffset;
    this._connected = false;
    this._playing = false;
  }

  get handyKey(): string {
    return this._handy.connectionKey;
  }

  async uploadScript(funscriptPath: string) {
    if (!(this._handy.connectionKey && funscriptPath)) {
      return;
    }

    if (!this._handy.serverTimeOffset) {
      const cachedOffset = localStorage.getItem("serverTimeOffset");
      if (cachedOffset !== null) {
        this._handy.serverTimeOffset = parseInt(cachedOffset, 10);
      } else {
        // One time sync to get server time offset
        await this._handy.getServerTimeOffset();
        localStorage.setItem(
          "serverTimeOffset",
          this._handy.serverTimeOffset.toString()
        );
      }
    }

    const csv = await fetch(funscriptPath)
      .then((response) => response.json())
      .then((json) => convertFunscriptToCSV(json));
    const fileName = `${Math.round(Math.random() * 100000000)}.csv`;
    const csvFile = new File([csv], fileName);
    const tempURL = await this._handy
      .uploadCsv(csvFile)
      .then((response) => response.url);
    this._connected = await this._handy
      .syncPrepare(encodeURIComponent(tempURL), fileName, csvFile.size)
      .then((response) => response.connected);
  }

  async play(position: number) {
    if (!this._connected) {
      return;
    }
    this._playing = await this._handy
      .syncPlay(true, Math.round(position * 1000 + this._scriptOffset))
      .then(() => true);
  }

  async pause() {
    if (!this._connected) {
      return;
    }
    this._playing = await this._handy.syncPlay(false).then(() => false);
  }

  async ensurePlaying(position: number) {
    if (this._playing) {
      return;
    }
    await this.play(position);
  }
}
