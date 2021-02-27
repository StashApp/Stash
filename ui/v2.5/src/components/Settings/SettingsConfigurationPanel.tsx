import React, { useEffect, useState } from "react";
import { Button, Form, InputGroup } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useConfiguration, useConfigureGeneral } from "src/core/StashService";
import { useToast } from "src/hooks";
import { Icon, LoadingIndicator } from "src/components/Shared";
import StashBoxConfiguration, {
  IStashBoxInstance,
} from "./StashBoxConfiguration";
import StashConfiguration from "./StashConfiguration";

interface IExclusionPatternsProps {
  excludes: string[];
  setExcludes: (value: string[]) => void;
}

const ExclusionPatterns: React.FC<IExclusionPatternsProps> = (props) => {
  function excludeRegexChanged(idx: number, value: string) {
    const newExcludes = props.excludes.map((regex, i) => {
      const ret = idx !== i ? regex : value;
      return ret;
    });
    props.setExcludes(newExcludes);
  }

  function excludeRemoveRegex(idx: number) {
    const newExcludes = props.excludes.filter((_regex, i) => i !== idx);

    props.setExcludes(newExcludes);
  }

  function excludeAddRegex() {
    const demo = "sample\\.mp4$";
    const newExcludes = props.excludes.concat(demo);

    props.setExcludes(newExcludes);
  }

  return (
    <>
      <Form.Group>
        {props.excludes &&
          props.excludes.map((regexp, i) => (
            <InputGroup>
              <Form.Control
                className="col col-sm-6 text-input"
                value={regexp}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  excludeRegexChanged(i, e.currentTarget.value)
                }
              />
              <InputGroup.Append>
                <Button variant="danger" onClick={() => excludeRemoveRegex(i)}>
                  <Icon icon="minus" />
                </Button>
              </InputGroup.Append>
            </InputGroup>
          ))}
      </Form.Group>
      <Button className="minimal" onClick={() => excludeAddRegex()}>
        <Icon icon="plus" />
      </Button>
    </>
  );
};

export const SettingsConfigurationPanel: React.FC = () => {
  const Toast = useToast();
  // Editing config state
  const [stashes, setStashes] = useState<GQL.StashConfig[]>([]);
  const [databasePath, setDatabasePath] = useState<string | undefined>(
    undefined
  );
  const [generatedPath, setGeneratedPath] = useState<string | undefined>(
    undefined
  );
  const [cachePath, setCachePath] = useState<string | undefined>(undefined);
  const [calculateMD5, setCalculateMD5] = useState<boolean>(false);
  const [videoFileNamingAlgorithm, setVideoFileNamingAlgorithm] = useState<
    GQL.HashAlgorithm | undefined
  >(undefined);
  const [parallelTasks, setParallelTasks] = useState<number>(0);
  const [previewSegments, setPreviewSegments] = useState<number>(0);
  const [previewSegmentDuration, setPreviewSegmentDuration] = useState<number>(
    0
  );
  const [previewExcludeStart, setPreviewExcludeStart] = useState<
    string | undefined
  >(undefined);
  const [previewExcludeEnd, setPreviewExcludeEnd] = useState<
    string | undefined
  >(undefined);
  const [previewPreset, setPreviewPreset] = useState<string>(
    GQL.PreviewPreset.Slow
  );
  const [
    transcodeHardwareAcceleration,
    setTranscodeHardwareAcceleration,
  ] = useState(false);
  const [maxTranscodeSize, setMaxTranscodeSize] = useState<
    GQL.StreamingResolutionEnum | undefined
  >(undefined);
  const [maxStreamingTranscodeSize, setMaxStreamingTranscodeSize] = useState<
    GQL.StreamingResolutionEnum | undefined
  >(undefined);
  const [username, setUsername] = useState<string | undefined>(undefined);
  const [password, setPassword] = useState<string | undefined>(undefined);
  const [maxSessionAge, setMaxSessionAge] = useState<number>(0);
  const [logFile, setLogFile] = useState<string | undefined>();
  const [logOut, setLogOut] = useState<boolean>(true);
  const [logLevel, setLogLevel] = useState<string>("Info");
  const [logAccess, setLogAccess] = useState<boolean>(true);

  const [videoExtensions, setVideoExtensions] = useState<string | undefined>();
  const [imageExtensions, setImageExtensions] = useState<string | undefined>();
  const [galleryExtensions, setGalleryExtensions] = useState<
    string | undefined
  >();
  const [
    createGalleriesFromFolders,
    setCreateGalleriesFromFolders,
  ] = useState<boolean>(false);

  const [excludes, setExcludes] = useState<string[]>([]);
  const [imageExcludes, setImageExcludes] = useState<string[]>([]);
  const [scraperUserAgent, setScraperUserAgent] = useState<string | undefined>(
    undefined
  );
  const [scraperCDPPath, setScraperCDPPath] = useState<string | undefined>(
    undefined
  );
  const [stashBoxes, setStashBoxes] = useState<IStashBoxInstance[]>([]);

  const { data, error, loading } = useConfiguration();

  const [updateGeneralConfig] = useConfigureGeneral({
    stashes: stashes.map((s) => ({
      path: s.path,
      excludeVideo: s.excludeVideo,
      excludeImage: s.excludeImage,
    })),
    databasePath,
    generatedPath,
    cachePath,
    calculateMD5,
    videoFileNamingAlgorithm:
      (videoFileNamingAlgorithm as GQL.HashAlgorithm) ?? undefined,
    parallelTasks,
    previewSegments,
    previewSegmentDuration,
    previewExcludeStart,
    previewExcludeEnd,
    previewPreset: (previewPreset as GQL.PreviewPreset) ?? undefined,
    transcodeHardwareAcceleration,
    maxTranscodeSize,
    maxStreamingTranscodeSize,
    username,
    password,
    maxSessionAge,
    logFile,
    logOut,
    logLevel,
    logAccess,
    createGalleriesFromFolders,
    videoExtensions: commaDelimitedToList(videoExtensions),
    imageExtensions: commaDelimitedToList(imageExtensions),
    galleryExtensions: commaDelimitedToList(galleryExtensions),
    excludes,
    imageExcludes,
    scraperUserAgent,
    scraperCDPPath,
    stashBoxes: stashBoxes.map(
      (b) =>
        ({
          name: b?.name ?? "",
          api_key: b?.api_key ?? "",
          endpoint: b?.endpoint ?? "",
        } as GQL.StashBoxInput)
    ),
  });

  useEffect(() => {
    if (!data?.configuration || error) return;

    const conf = data.configuration;
    if (conf.general) {
      setStashes(conf.general.stashes ?? []);
      setDatabasePath(conf.general.databasePath);
      setGeneratedPath(conf.general.generatedPath);
      setCachePath(conf.general.cachePath);
      setVideoFileNamingAlgorithm(conf.general.videoFileNamingAlgorithm);
      setCalculateMD5(conf.general.calculateMD5);
      setParallelTasks(conf.general.parallelTasks);
      setPreviewSegments(conf.general.previewSegments);
      setPreviewSegmentDuration(conf.general.previewSegmentDuration);
      setPreviewExcludeStart(conf.general.previewExcludeStart);
      setPreviewExcludeEnd(conf.general.previewExcludeEnd);
      setPreviewPreset(conf.general.previewPreset);
      setTranscodeHardwareAcceleration(
        conf.general.transcodeHardwareAcceleration
      );
      setMaxTranscodeSize(conf.general.maxTranscodeSize ?? undefined);
      setMaxStreamingTranscodeSize(
        conf.general.maxStreamingTranscodeSize ?? undefined
      );
      setUsername(conf.general.username);
      setPassword(conf.general.password);
      setMaxSessionAge(conf.general.maxSessionAge);
      setLogFile(conf.general.logFile ?? undefined);
      setLogOut(conf.general.logOut);
      setLogLevel(conf.general.logLevel);
      setLogAccess(conf.general.logAccess);
      setCreateGalleriesFromFolders(conf.general.createGalleriesFromFolders);
      setVideoExtensions(listToCommaDelimited(conf.general.videoExtensions));
      setImageExtensions(listToCommaDelimited(conf.general.imageExtensions));
      setGalleryExtensions(
        listToCommaDelimited(conf.general.galleryExtensions)
      );
      setExcludes(conf.general.excludes);
      setImageExcludes(conf.general.imageExcludes);
      setScraperUserAgent(conf.general.scraperUserAgent ?? undefined);
      setScraperCDPPath(conf.general.scraperCDPPath ?? undefined);
      setStashBoxes(
        conf.general.stashBoxes.map((box, i) => ({
          name: box?.name ?? undefined,
          endpoint: box.endpoint,
          api_key: box.api_key,
          index: i,
        })) ?? []
      );
    }
  }, [data, error]);

  function commaDelimitedToList(value: string | undefined) {
    if (value) {
      return value.split(",").map((s) => s.trim());
    }
  }

  function listToCommaDelimited(value: string[] | undefined) {
    if (value) {
      return value.join(", ");
    }
  }

  async function onSave() {
    try {
      const result = await updateGeneralConfig();
      // eslint-disable-next-line no-console
      console.log(result);
      Toast.success({ content: "Updated config" });
    } catch (e) {
      Toast.error(e);
    }
  }

  const transcodeQualities = [
    GQL.StreamingResolutionEnum.Low,
    GQL.StreamingResolutionEnum.Standard,
    GQL.StreamingResolutionEnum.StandardHd,
    GQL.StreamingResolutionEnum.FullHd,
    GQL.StreamingResolutionEnum.FourK,
    GQL.StreamingResolutionEnum.Original,
  ].map(resolutionToString);

  function resolutionToString(r: GQL.StreamingResolutionEnum | undefined) {
    switch (r) {
      case GQL.StreamingResolutionEnum.Low:
        return "240p";
      case GQL.StreamingResolutionEnum.Standard:
        return "480p";
      case GQL.StreamingResolutionEnum.StandardHd:
        return "720p";
      case GQL.StreamingResolutionEnum.FullHd:
        return "1080p";
      case GQL.StreamingResolutionEnum.FourK:
        return "4k";
      case GQL.StreamingResolutionEnum.Original:
        return "Original";
    }

    return "Original";
  }

  function translateQuality(quality: string) {
    switch (quality) {
      case "240p":
        return GQL.StreamingResolutionEnum.Low;
      case "480p":
        return GQL.StreamingResolutionEnum.Standard;
      case "720p":
        return GQL.StreamingResolutionEnum.StandardHd;
      case "1080p":
        return GQL.StreamingResolutionEnum.FullHd;
      case "4k":
        return GQL.StreamingResolutionEnum.FourK;
      case "Original":
        return GQL.StreamingResolutionEnum.Original;
    }

    return GQL.StreamingResolutionEnum.Original;
  }

  const namingHashAlgorithms = [
    GQL.HashAlgorithm.Md5,
    GQL.HashAlgorithm.Oshash,
  ].map(namingHashToString);

  function namingHashToString(value: GQL.HashAlgorithm | undefined) {
    switch (value) {
      case GQL.HashAlgorithm.Oshash:
        return "oshash";
      case GQL.HashAlgorithm.Md5:
        return "MD5";
    }

    return "MD5";
  }

  function translateNamingHash(value: string) {
    switch (value) {
      case "oshash":
        return GQL.HashAlgorithm.Oshash;
      case "MD5":
        return GQL.HashAlgorithm.Md5;
    }

    return GQL.HashAlgorithm.Md5;
  }

  if (error) return <h1>{error.message}</h1>;
  if (!data?.configuration || loading) return <LoadingIndicator />;

  return (
    <>
      <h4>Library</h4>
      <Form.Group>
        <Form.Group id="stashes">
          <h6>Stashes</h6>
          <StashConfiguration
            stashes={stashes}
            setStashes={(s) => setStashes(s)}
          />
          <Form.Text className="text-muted">
            Directory locations to your content
          </Form.Text>
        </Form.Group>

        <Form.Group id="database-path">
          <h6>Database Path</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={databasePath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setDatabasePath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            File location for the SQLite database (requires restart)
          </Form.Text>
        </Form.Group>

        <Form.Group id="generated-path">
          <h6>Generated Path</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={generatedPath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setGeneratedPath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Directory location for the generated files (scene markers, scene
            previews, sprites, etc)
          </Form.Text>
        </Form.Group>

        <Form.Group id="cache-path">
          <h6>Cache Path</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={cachePath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setCachePath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Directory location of the cache
          </Form.Text>
        </Form.Group>

        <Form.Group id="video-extensions">
          <h6>Video Extensions</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={videoExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setVideoExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Comma-delimited list of file extensions that will be identified as
            videos.
          </Form.Text>
        </Form.Group>

        <Form.Group id="image-extensions">
          <h6>Image Extensions</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={imageExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setImageExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Comma-delimited list of file extensions that will be identified as
            images.
          </Form.Text>
        </Form.Group>

        <Form.Group id="gallery-extensions">
          <h6>Gallery zip Extensions</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={galleryExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setGalleryExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Comma-delimited list of file extensions that will be identified as
            gallery zip files.
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <h6>Excluded Video Patterns</h6>
          <ExclusionPatterns excludes={excludes} setExcludes={setExcludes} />
          <Form.Text className="text-muted">
            Regexps of video files/paths to exclude from Scan and add to Clean
            <a
              href="https://github.com/stashapp/stash/wiki/Exclude-file-configuration"
              rel="noopener noreferrer"
              target="_blank"
            >
              <Icon icon="question-circle" />
            </a>
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <h6>Excluded Image/Gallery Patterns</h6>
          <ExclusionPatterns
            excludes={imageExcludes}
            setExcludes={setImageExcludes}
          />
          <Form.Text className="text-muted">
            Regexps of image and gallery files/paths to exclude from Scan and
            add to Clean
            <a
              href="https://github.com/stashapp/stash/wiki/Exclude-file-configuration"
              rel="noopener noreferrer"
              target="_blank"
            >
              <Icon icon="question-circle" />
            </a>
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <Form.Check
            id="log-terminal"
            checked={createGalleriesFromFolders}
            label="Create galleries from folders containing images"
            onChange={() =>
              setCreateGalleriesFromFolders(!createGalleriesFromFolders)
            }
          />
          <Form.Text className="text-muted">
            If true, creates galleries from folders containing images.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>Hashing</h4>
        <Form.Group>
          <Form.Check
            checked={calculateMD5}
            label="Calculate MD5 for videos"
            onChange={() => setCalculateMD5(!calculateMD5)}
          />
          <Form.Text className="text-muted">
            Calculate MD5 checksum in addition to oshash. Enabling will cause
            initial scans to be slower. File naming hash must be set to oshash
            to disable MD5 calculation.
          </Form.Text>
        </Form.Group>

        <Form.Group id="transcode-size">
          <h6>Generated file naming hash</h6>

          <Form.Control
            className="w-auto input-control"
            as="select"
            value={namingHashToString(videoFileNamingAlgorithm)}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
              setVideoFileNamingAlgorithm(
                translateNamingHash(e.currentTarget.value)
              )
            }
          >
            {namingHashAlgorithms.map((q) => (
              <option key={q} value={q}>
                {q}
              </option>
            ))}
          </Form.Control>

          <Form.Text className="text-muted">
            Use MD5 or oshash for generated file naming. Changing this requires
            that all scenes have the applicable MD5/oshash value populated.
            After changing this value, existing generated files will need to be
            migrated or regenerated. See Tasks page for migration.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>Video</h4>
        <Form.Group>
          <h6>Transcode Hardware Acceleration</h6>
          <Form.Check
            label="Enable hardware acceleration"
            id="transcode-hardware-acceleration"
            checked={transcodeHardwareAcceleration}
            onChange={(e) =>
              setTranscodeHardwareAcceleration(e.currentTarget.checked)
            }
          />
          <Form.Text className="text-muted">
            If enabled Stash will attempt to use hardware acceleration when live
            transcoding video streams.
          </Form.Text>
        </Form.Group>
        <Form.Group id="transcode-size">
          <h6>Maximum transcode size</h6>
          <Form.Control
            className="w-auto input-control"
            as="select"
            onChange={(event: React.ChangeEvent<HTMLSelectElement>) =>
              setMaxTranscodeSize(translateQuality(event.currentTarget.value))
            }
            value={resolutionToString(maxTranscodeSize)}
          >
            {transcodeQualities.map((q) => (
              <option key={q} value={q}>
                {q}
              </option>
            ))}
          </Form.Control>
          <Form.Text className="text-muted">
            Maximum size for generated transcodes
          </Form.Text>
        </Form.Group>
        <Form.Group id="streaming-transcode-size">
          <h6>Maximum streaming transcode size</h6>
          <Form.Control
            className="w-auto input-control"
            as="select"
            onChange={(event: React.ChangeEvent<HTMLSelectElement>) =>
              setMaxStreamingTranscodeSize(
                translateQuality(event.currentTarget.value)
              )
            }
            value={resolutionToString(maxStreamingTranscodeSize)}
          >
            {transcodeQualities.map((q) => (
              <option key={q} value={q}>
                {q}
              </option>
            ))}
          </Form.Control>
          <Form.Text className="text-muted">
            Maximum size for transcoded streams
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>Parallel Scan/Generation</h4>

        <Form.Group id="parallel-tasks">
          <h6>Number of parallel task for scan/generation</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="number"
            value={parallelTasks}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setParallelTasks(
                Number.parseInt(e.currentTarget.value || "0", 10)
              )
            }
          />
          <Form.Text className="text-muted">
            Set to 0 for auto-detection. Warning running more tasks than is
            required to achieve 100% cpu utilisation will decrease performance
            and potentially cause other issues.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>Preview Generation</h4>

        <Form.Group id="transcode-size">
          <h6>Preview encoding preset</h6>
          <Form.Control
            className="w-auto input-control"
            as="select"
            value={previewPreset}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
              setPreviewPreset(e.currentTarget.value)
            }
          >
            {Object.keys(GQL.PreviewPreset).map((p) => (
              <option value={p.toLowerCase()} key={p}>
                {p}
              </option>
            ))}
          </Form.Control>
          <Form.Text className="text-muted">
            The preset regulates size, quality and encoding time of preview
            generation. Presets beyond “slow” have diminishing returns and are
            not recommended.
          </Form.Text>
        </Form.Group>
        <Form.Group id="preview-segments">
          <h6>Number of segments in preview</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="number"
            value={previewSegments.toString()}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewSegments(
                Number.parseInt(e.currentTarget.value || "0", 10)
              )
            }
          />
          <Form.Text className="text-muted">
            Number of segments in preview files.
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-segment-duration">
          <h6>Preview segment duration</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="number"
            value={previewSegmentDuration.toString()}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewSegmentDuration(
                Number.parseFloat(e.currentTarget.value || "0")
              )
            }
          />
          <Form.Text className="text-muted">
            Duration of each preview segment, in seconds.
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-exclude-start">
          <h6>Exclude start time</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={previewExcludeStart}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewExcludeStart(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Exclude the first x seconds from scene previews. This can be a value
            in seconds, or a percentage (eg 2%) of the total scene duration.
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-exclude-start">
          <h6>Exclude end time</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={previewExcludeEnd}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewExcludeEnd(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Exclude the last x seconds from scene previews. This can be a value
            in seconds, or a percentage (eg 2%) of the total scene duration.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <Form.Group>
        <h4>Scraping</h4>
        <Form.Group id="scraperUserAgent">
          <h6>Scraper User Agent</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={scraperUserAgent}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setScraperUserAgent(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            User-Agent string used during scrape http requests
          </Form.Text>
        </Form.Group>

        <Form.Group id="scraperCDPPath">
          <h6>Chrome CDP path</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={scraperCDPPath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setScraperCDPPath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            File path to the Chrome executable, or a remote address (starting
            with http:// or https://, for example
            http://localhost:9222/json/version) to a Chrome instance.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group id="stashbox">
        <h4>Stash-box integration</h4>
        <StashBoxConfiguration boxes={stashBoxes} saveBoxes={setStashBoxes} />
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>Authentication</h4>
        <Form.Group id="username">
          <h6>Username</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={username}
            onInput={(e: React.FormEvent<HTMLInputElement>) =>
              setUsername(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Username to access Stash. Leave blank to disable user authentication
          </Form.Text>
        </Form.Group>
        <Form.Group id="password">
          <h6>Password</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="password"
            defaultValue={password}
            onInput={(e: React.FormEvent<HTMLInputElement>) =>
              setPassword(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            Password to access Stash. Leave blank to disable user authentication
          </Form.Text>
        </Form.Group>

        <Form.Group id="maxSessionAge">
          <h6>Maximum Session Age</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="number"
            value={maxSessionAge.toString()}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setMaxSessionAge(
                Number.parseInt(e.currentTarget.value || "0", 10)
              )
            }
          />
          <Form.Text className="text-muted">
            Maximum idle time before a login session is expired, in seconds.
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <h4>Logging</h4>
      <Form.Group id="log-file">
        <h6>Log file</h6>
        <Form.Control
          className="col col-sm-6 text-input"
          defaultValue={logFile}
          onInput={(e: React.FormEvent<HTMLInputElement>) =>
            setLogFile(e.currentTarget.value)
          }
        />
        <Form.Text className="text-muted">
          Path to the file to output logging to. Blank to disable file logging.
          Requires restart.
        </Form.Text>
      </Form.Group>

      <Form.Group>
        <Form.Check
          id="log-terminal"
          checked={logOut}
          label="Log to terminal"
          onChange={() => setLogOut(!logOut)}
        />
        <Form.Text className="text-muted">
          Logs to the terminal in addition to a file. Always true if file
          logging is disabled. Requires restart.
        </Form.Text>
      </Form.Group>

      <Form.Group id="log-level">
        <h6>Log Level</h6>
        <Form.Control
          className="col col-sm-6 input-control"
          as="select"
          onChange={(event: React.ChangeEvent<HTMLSelectElement>) =>
            setLogLevel(event.currentTarget.value)
          }
          value={logLevel}
        >
          {["Trace", "Debug", "Info", "Warning", "Error"].map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </Form.Control>
      </Form.Group>

      <Form.Group>
        <Form.Check
          id="log-http"
          checked={logAccess}
          label="Log http access"
          onChange={() => setLogAccess(!logAccess)}
        />
        <Form.Text className="text-muted">
          Logs http access to the terminal. Requires restart.
        </Form.Text>
      </Form.Group>

      <hr />

      <Button variant="primary" onClick={() => onSave()}>
        Save
      </Button>
    </>
  );
};
