import React, { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button } from "react-bootstrap";
import {
  mutateReloadScrapers,
  useListMovieScrapers,
  useListPerformerScrapers,
  useListSceneScrapers,
  useListGalleryScrapers,
} from "src/core/StashService";
import { useToast } from "src/hooks";
import { TextUtils } from "src/utils";
import { CollapseButton, Icon, LoadingIndicator } from "src/components/Shared";
import { ScrapeType } from "src/core/generated-graphql";

interface IURLList {
  urls: string[];
}

const URLList: React.FC<IURLList> = ({ urls }) => {
  const maxCollapsedItems = 5;
  const [expanded, setExpanded] = useState<boolean>(false);

  function linkSite(url: string) {
    const u = new URL(url);
    return `${u.protocol}//${u.host}`;
  }

  function renderLink(url?: string) {
    if (url) {
      const sanitised = TextUtils.sanitiseURL(url);
      const siteURL = linkSite(sanitised!);

      return (
        <a
          href={siteURL}
          className="link"
          target="_blank"
          rel="noopener noreferrer"
        >
          {sanitised}
        </a>
      );
    }
  }

  function getListItems() {
    const items = urls.map((u) => <li key={u}>{renderLink(u)}</li>);

    if (items.length > maxCollapsedItems) {
      if (!expanded) {
        items.length = maxCollapsedItems;
      }

      items.push(
        <li key="expand/collapse">
          <Button onClick={() => setExpanded(!expanded)} variant="link">
            {expanded ? "less" : "more"}
          </Button>
        </li>
      );
    }

    return items;
  }

  return <ul>{getListItems()}</ul>;
};

export const SettingsScrapersPanel: React.FC = () => {
  const Toast = useToast();
  const intl = useIntl();
  const {
    data: performerScrapers,
    loading: loadingPerformers,
  } = useListPerformerScrapers();
  const {
    data: sceneScrapers,
    loading: loadingScenes,
  } = useListSceneScrapers();
  const {
    data: galleryScrapers,
    loading: loadingGalleries,
  } = useListGalleryScrapers();
  const {
    data: movieScrapers,
    loading: loadingMovies,
  } = useListMovieScrapers();

  async function onReloadScrapers() {
    await mutateReloadScrapers().catch((e) => Toast.error(e));
  }

  function renderPerformerScrapeTypes(types: ScrapeType[]) {
    const typeStrings = types
      .filter((t) => t !== ScrapeType.Fragment)
      .map((t) => {
        switch (t) {
          case ScrapeType.Name:
            return intl.formatMessage({ id: "config.scrapers.search_by_name" });
          default:
            return t;
        }
      });

    return (
      <ul>
        {typeStrings.map((t) => (
          <li key={t}>{t}</li>
        ))}
      </ul>
    );
  }

  function renderSceneScrapeTypes(types: ScrapeType[]) {
    const typeStrings = types.map((t) => {
      switch (t) {
        case ScrapeType.Fragment:
          return intl.formatMessage(
            { id: "config.scrapers.entity_metadata" },
            { entityType: intl.formatMessage({ id: "scene" }) }
          );
        default:
          return t;
      }
    });

    return (
      <ul>
        {typeStrings.map((t) => (
          <li key={t}>{t}</li>
        ))}
      </ul>
    );
  }

  function renderGalleryScrapeTypes(types: ScrapeType[]) {
    const typeStrings = types.map((t) => {
      switch (t) {
        case ScrapeType.Fragment:
          return intl.formatMessage(
            { id: "config.scrapers.entity_metadata" },
            { entityType: intl.formatMessage({ id: "gallery" }) }
          );
        default:
          return t;
      }
    });

    return (
      <ul>
        {typeStrings.map((t) => (
          <li key={t}>{t}</li>
        ))}
      </ul>
    );
  }

  function renderMovieScrapeTypes(types: ScrapeType[]) {
    const typeStrings = types.map((t) => {
      switch (t) {
        case ScrapeType.Fragment:
          return intl.formatMessage(
            { id: "config.scrapers.entity_metadata" },
            { entityType: intl.formatMessage({ id: "movie" }) }
          );
        default:
          return t;
      }
    });

    return (
      <ul>
        {typeStrings.map((t) => (
          <li key={t}>{t}</li>
        ))}
      </ul>
    );
  }

  function renderURLs(urls: string[]) {
    return <URLList urls={urls} />;
  }

  function renderSceneScrapers() {
    const elements = (sceneScrapers?.listSceneScrapers ?? []).map((scraper) => (
      <tr key={scraper.id}>
        <td>{scraper.name}</td>
        <td>
          {renderSceneScrapeTypes(scraper.scene?.supported_scrapes ?? [])}
        </td>
        <td>{renderURLs(scraper.scene?.urls ?? [])}</td>
      </tr>
    ));

    return renderTable(
      intl.formatMessage(
        { id: "config.scrapers.entity_scrapers" },
        { entityType: intl.formatMessage({ id: "scene" }) }
      ),
      elements
    );
  }

  function renderGalleryScrapers() {
    const elements = (galleryScrapers?.listGalleryScrapers ?? []).map(
      (scraper) => (
        <tr key={scraper.id}>
          <td>{scraper.name}</td>
          <td>
            {renderGalleryScrapeTypes(scraper.gallery?.supported_scrapes ?? [])}
          </td>
          <td>{renderURLs(scraper.gallery?.urls ?? [])}</td>
        </tr>
      )
    );

    return renderTable(
      intl.formatMessage(
        { id: "config.scrapers.entity_scrapers" },
        { entityType: intl.formatMessage({ id: "gallery" }) }
      ),
      elements
    );
  }

  function renderPerformerScrapers() {
    const elements = (performerScrapers?.listPerformerScrapers ?? []).map(
      (scraper) => (
        <tr key={scraper.id}>
          <td>{scraper.name}</td>
          <td>
            {renderPerformerScrapeTypes(
              scraper.performer?.supported_scrapes ?? []
            )}
          </td>
          <td>{renderURLs(scraper.performer?.urls ?? [])}</td>
        </tr>
      )
    );

    return renderTable(
      intl.formatMessage(
        { id: "config.scrapers.entity_scrapers" },
        { entityType: intl.formatMessage({ id: "performer" }) }
      ),
      elements
    );
  }

  function renderMovieScrapers() {
    const elements = (movieScrapers?.listMovieScrapers ?? []).map((scraper) => (
      <tr key={scraper.id}>
        <td>{scraper.name}</td>
        <td>
          {renderMovieScrapeTypes(scraper.movie?.supported_scrapes ?? [])}
        </td>
        <td>{renderURLs(scraper.movie?.urls ?? [])}</td>
      </tr>
    ));

    return renderTable(
      intl.formatMessage(
        { id: "config.scrapers.entity_scrapers" },
        { entityType: intl.formatMessage({ id: "movie" }) }
      ),
      elements
    );
  }

  function renderTable(title: string, elements: JSX.Element[]) {
    if (elements.length > 0) {
      return (
        <CollapseButton text={title}>
          <table className="scraper-table">
            <thead>
              <tr>
                <th>{intl.formatMessage({ id: "name" })}</th>
                <th>
                  {intl.formatMessage({
                    id: "config.scrapers.supported_types",
                  })}
                </th>
                <th>
                  {intl.formatMessage({ id: "config.scrapers.supported_urls" })}
                </th>
              </tr>
            </thead>
            <tbody>{elements}</tbody>
          </table>
        </CollapseButton>
      );
    }
  }

  if (loadingScenes || loadingGalleries || loadingPerformers || loadingMovies)
    return <LoadingIndicator />;

  return (
    <>
      <h4>{intl.formatMessage({ id: "config.categories.scrapers" })}</h4>
      <div className="mb-3">
        <Button onClick={() => onReloadScrapers()}>
          <span className="fa-icon">
            <Icon icon="sync-alt" />
          </span>
          <span>
            <FormattedMessage id="actions.reload_scrapers" />
          </span>
        </Button>
      </div>

      <div>
        {renderSceneScrapers()}
        {renderGalleryScrapers()}
        {renderPerformerScrapers()}
        {renderMovieScrapers()}
      </div>
    </>
  );
};
