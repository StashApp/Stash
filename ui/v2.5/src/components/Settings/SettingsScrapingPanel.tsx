import React, { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button, Form } from "react-bootstrap";
import {
  mutateReloadScrapers,
  useListMovieScrapers,
  useListPerformerScrapers,
  useListSceneScrapers,
  useListGalleryScrapers,
  useConfiguration,
  useConfigureScraping,
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

export const SettingsScrapingPanel: React.FC = () => {
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

  const [scraperUserAgent, setScraperUserAgent] = useState<string | undefined>(
    undefined
  );
  const [scraperCDPPath, setScraperCDPPath] = useState<string | undefined>(
    undefined
  );
  const [scraperCertCheck, setScraperCertCheck] = useState<boolean>(true);

  const { data, error } = useConfiguration();

  const [updateScrapingConfig] = useConfigureScraping({
    scraperUserAgent,
    scraperCDPPath,
    scraperCertCheck,
  });

  useEffect(() => {
    if (!data?.configuration || error) return;

    const conf = data.configuration;
    if (conf.scraping) {
      setScraperUserAgent(conf.scraping.scraperUserAgent ?? undefined);
      setScraperCDPPath(conf.scraping.scraperCDPPath ?? undefined);
      setScraperCertCheck(conf.scraping.scraperCertCheck);
    }
  }, [data, error]);

  async function onReloadScrapers() {
    await mutateReloadScrapers().catch((e) => Toast.error(e));
  }

  async function onSave() {
    try {
      await updateScrapingConfig();
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

  function renderPerformerScrapeTypes(types: ScrapeType[]) {
    const typeStrings = types
      .filter((t) => t !== ScrapeType.Fragment)
      .map((t) => {
        switch (t) {
          case ScrapeType.Name:
            return intl.formatMessage({ id: "config.scraping.search_by_name" });
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
            { id: "config.scraping.entity_metadata" },
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
            { id: "config.scraping.entity_metadata" },
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
            { id: "config.scraping.entity_metadata" },
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
        { id: "config.scraping.entity_scrapers" },
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
        { id: "config.scraping.entity_scrapers" },
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
        { id: "config.scraping.entity_scrapers" },
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
        { id: "config.scraping.entity_scrapers" },
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
                    id: "config.scraping.supported_types",
                  })}
                </th>
                <th>
                  {intl.formatMessage({ id: "config.scraping.supported_urls" })}
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
      <Form.Group>
        <h4>{intl.formatMessage({ id: "config.general.scraping" })}</h4>
        <Form.Group id="scraperUserAgent">
          <h6>
            {intl.formatMessage({ id: "config.general.scraper_user_agent" })}
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={scraperUserAgent}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setScraperUserAgent(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.scraper_user_agent_desc",
            })}
          </Form.Text>
        </Form.Group>

        <Form.Group id="scraperCDPPath">
          <h6>
            {intl.formatMessage({ id: "config.general.chrome_cdp_path" })}
          </h6>
          <Form.Control
            className="col col-sm-6 text-input"
            defaultValue={scraperCDPPath}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setScraperCDPPath(e.currentTarget.value)
            }
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({ id: "config.general.chrome_cdp_path_desc" })}
          </Form.Text>
        </Form.Group>

        <Form.Group>
          <Form.Check
            id="scaper-cert-check"
            checked={scraperCertCheck}
            label={intl.formatMessage({
              id: "config.general.check_for_insecure_certificates",
            })}
            onChange={() => setScraperCertCheck(!scraperCertCheck)}
          />
          <Form.Text className="text-muted">
            {intl.formatMessage({
              id: "config.general.check_for_insecure_certificates_desc",
            })}
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <hr />

      <h4>{intl.formatMessage({ id: "config.scraping.scrapers" })}</h4>

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

      <hr />

      <Button variant="primary" onClick={() => onSave()}>
        <FormattedMessage id="actions.save" />
      </Button>
    </>
  );
};
