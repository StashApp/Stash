import React, { useRef, useState } from "react";
import { Button, Form, InputGroup } from "react-bootstrap";
import { Link } from "react-router-dom";
import { FormattedMessage } from "react-intl";
import { ScenePreview } from "src/components/Scenes/SceneCard";

import * as GQL from "src/core/generated-graphql";
import { TruncatedText } from "src/components/Shared";
import StashSearchResult from "./StashSearchResult";
import { ITaggerConfig } from "./constants";
import {
  parsePath,
  IStashBoxScene,
  sortScenesByDuration,
  prepareQueryString,
} from "./utils";

export interface ISearchResult {
  results?: IStashBoxScene[];
  error?: string;
}

export interface ITaggerScene {
  scene: GQL.SlimSceneDataFragment;
  url: string;
  config: ITaggerConfig;
  searchResult?: ISearchResult;
  hideUnmatched?: boolean;
  loading?: boolean;
  doSceneQuery: (queryString: string) => void;
  taggedScene?: Partial<GQL.SlimSceneDataFragment>;
  tagScene: (scene: Partial<GQL.SlimSceneDataFragment>) => void;
  endpoint: string;
  queueFingerprintSubmission: (sceneId: string, endpoint: string) => void;
}

export const TaggerScene: React.FC<ITaggerScene> = ({
  scene,
  url,
  config,
  searchResult,
  hideUnmatched,
  loading,
  doSceneQuery,
  taggedScene,
  tagScene,
  endpoint,
  queueFingerprintSubmission,
}) => {
  const [selectedResult, setSelectedResult] = useState<number>(0);

  const queryString = useRef<string>("");

  const searchResults = searchResult?.results ?? [];
  const searchError = searchResult?.error;
  const emptyResults =
    searchResult && searchResult.results && searchResult.results.length === 0;

  const { paths, file, ext } = parsePath(scene.path);
  const originalDir = scene.path.slice(
    0,
    scene.path.length - file.length - ext.length
  );
  const defaultQueryString = prepareQueryString(
    scene,
    paths,
    file,
    config.mode,
    config.blacklist
  );

  const hasStashIDs = scene.stash_ids.length > 0;
  const width = scene.file.width ? scene.file.width : 0;
  const height = scene.file.height ? scene.file.height : 0;
  const isPortrait = height > width;

  function renderMainContent() {
    if (!taggedScene && hasStashIDs) {
      return (
        <div className="text-right">
          <h5 className="text-bold">
            <FormattedMessage id="component_tagger.results.match_failed_already_tagged" />
          </h5>
        </div>
      );
    }

    if (!taggedScene && !hasStashIDs) {
      return (
        <InputGroup>
          <InputGroup.Prepend>
            <InputGroup.Text>
              <FormattedMessage id="component_tagger.noun_query" />
            </InputGroup.Text>
          </InputGroup.Prepend>
          <Form.Control
            className="text-input"
            defaultValue={queryString.current || defaultQueryString}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              queryString.current = e.currentTarget.value;
            }}
            onKeyPress={(e: React.KeyboardEvent<HTMLInputElement>) =>
              e.key === "Enter" &&
              doSceneQuery(queryString.current || defaultQueryString)
            }
          />
          <InputGroup.Append>
            <Button
              disabled={loading}
              onClick={() =>
                doSceneQuery(queryString.current || defaultQueryString)
              }
            >
              <FormattedMessage id="actions.search" />
            </Button>
          </InputGroup.Append>
        </InputGroup>
      );
    }

    if (taggedScene) {
      return (
        <div className="d-flex flex-column text-right">
          <h5>
            <FormattedMessage id="component_tagger.results.match_success" />
          </h5>
          <h6>
            <Link className="bold" to={url}>
              {taggedScene.title}
            </Link>
          </h6>
        </div>
      );
    }
  }

  function renderSubContent() {
    if (scene.stash_ids.length > 0) {
      const stashLinks = scene.stash_ids.map((stashID) => {
        const base = stashID.endpoint.match(/https?:\/\/.*?\//)?.[0];
        const link = base ? (
          <a
            key={`${stashID.endpoint}${stashID.stash_id}`}
            className="small d-block"
            href={`${base}scenes/${stashID.stash_id}`}
            target="_blank"
            rel="noopener noreferrer"
          >
            {stashID.stash_id}
          </a>
        ) : (
          <div className="small">{stashID.stash_id}</div>
        );

        return link;
      });
      return <>{stashLinks}</>;
    }

    if (searchError) {
      return <div className="text-danger font-weight-bold">{searchError}</div>;
    }

    if (emptyResults) {
      return (
        <div className="text-danger font-weight-bold">
          <FormattedMessage id="component_tagger.results.match_failed_no_result" />
        </div>
      );
    }
  }

  function renderSearchResult() {
    if (searchResults.length > 0 && !taggedScene) {
      return (
        <ul className="pl-0 mt-3 mb-0">
          {sortScenesByDuration(
            searchResults,
            scene.file.duration ?? undefined
          ).map(
            (sceneResult, i) =>
              sceneResult && (
                <StashSearchResult
                  key={sceneResult.stash_id}
                  showMales={config.showMales}
                  stashScene={scene}
                  scene={sceneResult}
                  isActive={selectedResult === i}
                  setActive={() => setSelectedResult(i)}
                  setCoverImage={config.setCoverImage}
                  tagOperation={config.tagOperation}
                  setTags={config.setTags}
                  setScene={tagScene}
                  endpoint={endpoint}
                  queueFingerprintSubmission={queueFingerprintSubmission}
                />
              )
          )}
        </ul>
      );
    }
  }

  return hideUnmatched && emptyResults ? null : (
    <div key={scene.id} className="mt-3 search-item">
      <div className="row">
        <div className="col col-lg-6 overflow-hidden align-items-center d-flex flex-column flex-sm-row">
          <div className="scene-card mr-3">
            <Link to={url}>
              <ScenePreview
                image={scene.paths.screenshot ?? undefined}
                video={scene.paths.preview ?? undefined}
                isPortrait={isPortrait}
                soundActive={false}
              />
            </Link>
          </div>
          <Link to={url} className="scene-link overflow-hidden">
            <TruncatedText
              text={`${originalDir}\u200B${file}${ext}`}
              lineCount={2}
            />
          </Link>
        </div>
        <div className="col-md-6 my-1 align-self-center">
          {renderMainContent()}
          <div className="sub-content text-right">{renderSubContent()}</div>
        </div>
      </div>
      {renderSearchResult()}
    </div>
  );
};
