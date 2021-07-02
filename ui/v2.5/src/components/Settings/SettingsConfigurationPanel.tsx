import React, { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button, Form, InputGroup } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import {
  useConfiguration,
  useConfigureGeneral,
  useGenerateAPIKey,
} from "src/core/StashService";
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
  const intl = useIntl();
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
  const [previewAudio, setPreviewAudio] = useState<boolean>(true);
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
  const [stashBoxes, setStashBoxes] = useState<IStashBoxInstance[]>([]);

  const { data, error, loading } = useConfiguration();

  const [generateAPIKey] = useGenerateAPIKey();

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
    previewAudio,
    previewSegments,
    previewSegmentDuration,
    previewExcludeStart,
    previewExcludeEnd,
    previewPreset: (previewPreset as GQL.PreviewPreset) ?? undefined,
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
      setPreviewAudio(conf.general.previewAudio);
      setPreviewSegments(conf.general.previewSegments);
      setPreviewSegmentDuration(conf.general.previewSegmentDuration);
      setPreviewExcludeStart(conf.general.previewExcludeStart);
      setPreviewExcludeEnd(conf.general.previewExcludeEnd);
      setPreviewPreset(conf.general.previewPreset);
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

  async function onGenerateAPIKey() {
    try {
      await generateAPIKey({
        variables: {
          input: {},
        },
      });
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onClearAPIKey() {
    try {
      await generateAPIKey({
        variables: {
          input: {
            clear: true,
          },
        },
      });
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onSave() {
    try {
      const result = await updateGeneralConfig();
      // eslint-disable-next-line no-console
      console.log(result);
      Toast.success({
        content: intl.formatMessage(
          { id: "toast.updated_entity" },
          {
            entity: intl
              .formatMessage({ id: "configuration" })
              .toLocaleLowerCase(),
          }
        ),
      });
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
      <h4>
        <FormattedMessage id="library" />
      </h4>
      <Form.Group>
        <Form.Group id="stashes">
          <h6>Stashes</h6>
          <StashConfiguration
            stashes={stashes}
            setStashes={(s) => setStashes(s)}
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.directory_locations_to_your_content",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="database-path">
          <h6>
            <FormattedMessage id="config.general.db_path_head" />
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={databasePath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setDatabasePath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.sqlite_location" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="generated-path">
          <h6>
            <FormattedMessage id="config.general.generated_path_head" />
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={generatedPath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setGeneratedPath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.generated_files_location",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="cache-path">
          <h6>
            <FormattedMessage id="config.general.cache_path_head" />
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={cachePath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setCachePath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.cache_location" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="video-extensions">
          <h6>
            <FormattedMessage id="config.general.video_ext_head" />
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={videoExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setVideoExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.video_ext_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="image-extensions">
          <h6>
            <FormattedMessage id="config.general.image_ext_head" />
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={imageExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setImageExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.image_ext_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="gallery-extensions">
          <h6>
            {intl.formatMessage({ id: "config.general.gallery_ext_head" })}
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={galleryExtensions}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setGalleryExtensions(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.gallery_ext_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <h6>
            {intl.formatMessage({
              id: "config.general.excluded_video_patterns_head",
            })}
          </h6>
          <ExclusionPatterns excludes={excludes} setExcludes={setExcludes} />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.excluded_video_patterns_desc",
            })}
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
          <h6>
            {intl.formatMessage({
              id: "config.general.excluded_image_gallery_patterns_head",
            })}
          </h6>
          <ExclusionPatterns
            excludes={imageExcludes}
            setExcludes={setImageExcludes}
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.excluded_image_gallery_patterns_desc",
            })}
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
            label={intl.formatMessage({
              id: "config.general.create_galleries_from_folders_label",
            })}
            onChange={() =>
              setCreateGalleriesFromFolders(!createGalleriesFromFolders)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.create_galleries_from_folders_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>{intl.formatMessage({ id: "config.general.hashing" })}</h4>
        <Form.Group>
          <Form.Check
            checked={calculateMD5}
            label={intl.formatMessage({
              id: "config.general.calculate_md5_and_ohash_label",
            })}
            onChange={() => setCalculateMD5(!calculateMD5)}
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.calculate_md5_and_ohash_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="transcode-size">
          <h6>
            {intl.formatMessage({
              id: "config.general.generated_file_naming_hash_head",
            })}
          </h6>

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
            {intl.formatMessage({
              id: "config.general.generated_file_naming_hash_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>{intl.formatMessage({ id: "config.general.video_head" })}</h4>
        <Form.Group id="transcode-size">
          <h6>
            {intl.formatMessage({
              id: "config.general.maximum_transcode_size_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "config.general.maximum_transcode_size_desc",
            })}
          </Form.Text>
        </Form.Group>
        <Form.Group id="streaming-transcode-size">
          <h6>
            {intl.formatMessage({
              id: "config.general.maximum_streaming_transcode_size_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "config.general.maximum_streaming_transcode_size_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>
          {intl.formatMessage({ id: "config.general.parallel_scan_head" })}
        </h4>

        <Form.Group id="parallel-tasks">
          <h6>
            {intl.formatMessage({
              id:
                "config.general.number_of_parallel_task_for_scan_generation_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id:
                "config.general.number_of_parallel_task_for_scan_generation_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>
          {intl.formatMessage({ id: "config.general.preview_generation" })}
        </h4>

        <Form.Group id="transcode-size">
          <h6>
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_preset_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_preset_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <Form.Check
            id="preview-include-audio"
            checked={previewAudio}
            label="Include audio"
            onChange={() => setPreviewAudio(!previewAudio)}
          />
          <Form.Text className="text-muted">
            Includes audio stream when generating previews.
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-segments">
          <h6>
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_seg_count_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_seg_count_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-segment-duration">
          <h6>
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_seg_duration_head",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_seg_duration_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-exclude-start">
          <h6>
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_exclude_start_time_head",
            })}
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={previewExcludeStart}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewExcludeStart(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_exclude_start_time_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="preview-exclude-start">
          <h6>
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_exclude_end_time_head",
            })}
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={previewExcludeEnd}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setPreviewExcludeEnd(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "dialogs.scene_gen.preview_exclude_end_time_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <Form.Group id="stashbox">
        <h4>
          {intl.formatMessage({
            id: "config.general.auth.stash-box_integration",
          })}
        </h4>
        <StashBoxConfiguration boxes={stashBoxes} saveBoxes={setStashBoxes} />
      </Form.Group>

      <hr />

      <Form.Group>
        <h4>
          {intl.formatMessage({ id: "config.general.auth.authentication" })}
        </h4>
        <Form.Group id="username">
          <h6>{intl.formatMessage({ id: "config.general.auth.username" })}</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={username}
            onInput={(e: React.FormEvent<HTMLInputElement>) =>
              setUsername(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.auth.username_desc" })}
          </Form.Text>
        </Form.Group>
        <Form.Group id="password">
          <h6>{intl.formatMessage({ id: "config.general.auth.password" })}</h6>
          <Form.Control
            className="col col-sm-6 text-input"
            type="password"
            defaultValue={password}
            onInput={(e: React.FormEvent<HTMLInputElement>) =>
              setPassword(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.auth.password_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="apikey">
          <h6>{intl.formatMessage({ id: "config.general.auth.api_key" })}</h6>
          <InputGroup>
            <Form.Control
              className="col col-sm-6 text-input"
              value={data.configuration.general.apiKey}
              readOnly
            />
            <InputGroup.Append>
              <Button
                className=""
                title={intl.formatMessage({
                  id: "config.general.auth.generate_api_key",
                })}
                onClick={() => onGenerateAPIKey()}
              >
                <Icon icon="redo" />
              </Button>
              <Button
                className=""
                variant="danger"
                title={intl.formatMessage({
                  id: "config.general.auth.clear_api_key",
                })}
                onClick={() => onClearAPIKey()}
              >
                <Icon icon="minus" />
              </Button>
            </InputGroup.Append>
          </InputGroup>
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.auth.api_key_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="maxSessionAge">
          <h6>
            {intl.formatMessage({
              id: "config.general.auth.maximum_session_age",
            })}
          </h6>
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
            {intl.formatMessage({
              id: "config.general.auth.maximum_session_age_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <h4>{intl.formatMessage({ id: "config.general.logging" })}</h4>
      <Form.Group id="log-file">
        <h6>{intl.formatMessage({ id: "config.general.auth.log_file" })}</h6>
        <Form.Control
          className="col col-sm-6 text-input"
          defaultValue={logFile}
          onInput={(e: React.FormEvent<HTMLInputElement>) =>
            setLogFile(e.currentTarget.value)
          }
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({ id: "config.general.auth.log_file_desc" })}
        </Form.Text>
      </Form.Group>

      <Form.Group>
        <Form.Check
          id="log-terminal"
          checked={logOut}
          label={intl.formatMessage({
            id: "config.general.auth.log_to_terminal",
          })}
          onChange={() => setLogOut(!logOut)}
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({
            id: "config.general.auth.log_to_terminal_desc",
          })}
        </Form.Text>
      </Form.Group>

      <Form.Group id="log-level">
        <h6>{intl.formatMessage({ id: "config.logs.log_level" })}</h6>
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
          label={intl.formatMessage({ id: "config.general.auth.log_http" })}
          onChange={() => setLogAccess(!logAccess)}
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({ id: "config.general.auth.log_http_desc" })}
        </Form.Text>
      </Form.Group>

      <hr />

      <Button variant="primary" onClick={() => onSave()}>
        <FormattedMessage id="actions.save" />
      </Button>
    </>
  );
};
