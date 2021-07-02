import { ApolloCache, DocumentNode } from "@apollo/client";
import {
  isField,
  resultKeyNameFromField,
  getQueryDefinition,
  getOperationName,
} from "@apollo/client/utilities";
import { stringToGender } from "src/utils/gender";
import { filterData } from "../utils";
import { ListFilterModel } from "../models/list-filter/filter";
import * as GQL from "./generated-graphql";

import { createClient } from "./createClient";

const { client } = createClient();

export const getClient = () => client;

const getQueryNames = (queries: DocumentNode[]): string[] =>
  queries.map((q) => getOperationName(q)).filter((n) => n !== null) as string[];

// Will delete the entire cache for any queries passed in
const deleteCache = (queries: DocumentNode[]) => {
  const fields = queries
    .map((q) => {
      const field = getQueryDefinition(q).selectionSet.selections[0];
      return isField(field) ? resultKeyNameFromField(field) : "";
    })
    .filter((name) => name !== "")
    .reduce(
      (prevFields, name) => ({
        ...prevFields,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        [name]: (_items: any, { DELETE }: any) => DELETE,
      }),
      {}
    );

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (cache: ApolloCache<any>) =>
    cache.modify({
      id: "ROOT_QUERY",
      fields,
    });
};

export const useFindSavedFilters = (mode: GQL.FilterMode) =>
  GQL.useFindSavedFiltersQuery({
    variables: {
      mode,
    },
  });

export const useFindDefaultFilter = (mode: GQL.FilterMode) =>
  GQL.useFindDefaultFilterQuery({
    variables: {
      mode,
    },
  });

export const useFindGalleries = (filter: ListFilterModel) =>
  GQL.useFindGalleriesQuery({
    variables: {
      filter: filter.makeFindFilter(),
      gallery_filter: filter.makeFilter(),
    },
  });

export const queryFindGalleries = (filter: ListFilterModel) =>
  client.query<GQL.FindGalleriesQuery>({
    query: GQL.FindGalleriesDocument,
    variables: {
      filter: filter.makeFindFilter(),
      gallery_filter: filter.makeFilter(),
    },
  });

export const useFindScenes = (filter: ListFilterModel) =>
  GQL.useFindScenesQuery({
    variables: {
      filter: filter.makeFindFilter(),
      scene_filter: filter.makeFilter(),
    },
  });

export const queryFindScenes = (filter: ListFilterModel) =>
  client.query<GQL.FindScenesQuery>({
    query: GQL.FindScenesDocument,
    variables: {
      filter: filter.makeFindFilter(),
      scene_filter: filter.makeFilter(),
    },
  });

export const queryFindScenesByID = (sceneIDs: number[]) =>
  client.query<GQL.FindScenesQuery>({
    query: GQL.FindScenesDocument,
    variables: {
      scene_ids: sceneIDs,
    },
  });

export const useFindSceneMarkers = (filter: ListFilterModel) =>
  GQL.useFindSceneMarkersQuery({
    variables: {
      filter: filter.makeFindFilter(),
      scene_marker_filter: filter.makeFilter(),
    },
  });

export const queryFindSceneMarkers = (filter: ListFilterModel) =>
  client.query<GQL.FindSceneMarkersQuery>({
    query: GQL.FindSceneMarkersDocument,
    variables: {
      filter: filter.makeFindFilter(),
      scene_marker_filter: filter.makeFilter(),
    },
  });

export const useFindImages = (filter: ListFilterModel) =>
  GQL.useFindImagesQuery({
    variables: {
      filter: filter.makeFindFilter(),
      image_filter: filter.makeFilter(),
    },
  });

export const queryFindImages = (filter: ListFilterModel) =>
  client.query<GQL.FindImagesQuery>({
    query: GQL.FindImagesDocument,
    variables: {
      filter: filter.makeFindFilter(),
      image_filter: filter.makeFilter(),
    },
  });

export const useFindStudios = (filter: ListFilterModel) =>
  GQL.useFindStudiosQuery({
    variables: {
      filter: filter.makeFindFilter(),
      studio_filter: filter.makeFilter(),
    },
  });

export const queryFindStudios = (filter: ListFilterModel) =>
  client.query<GQL.FindStudiosQuery>({
    query: GQL.FindStudiosDocument,
    variables: {
      filter: filter.makeFindFilter(),
      studio_filter: filter.makeFilter(),
    },
  });

export const useFindMovies = (filter: ListFilterModel) =>
  GQL.useFindMoviesQuery({
    variables: {
      filter: filter.makeFindFilter(),
      movie_filter: filter.makeFilter(),
    },
  });

export const queryFindMovies = (filter: ListFilterModel) =>
  client.query<GQL.FindMoviesQuery>({
    query: GQL.FindMoviesDocument,
    variables: {
      filter: filter.makeFindFilter(),
      movie_filter: filter.makeFilter(),
    },
  });

export const useFindPerformers = (filter: ListFilterModel) =>
  GQL.useFindPerformersQuery({
    variables: {
      filter: filter.makeFindFilter(),
      performer_filter: filter.makeFilter(),
    },
  });

export const useFindTags = (filter: ListFilterModel) =>
  GQL.useFindTagsQuery({
    variables: {
      filter: filter.makeFindFilter(),
      tag_filter: filter.makeFilter(),
    },
  });

export const queryFindTags = (filter: ListFilterModel) =>
  client.query<GQL.FindTagsQuery>({
    query: GQL.FindTagsDocument,
    variables: {
      filter: filter.makeFindFilter(),
      tag_filter: filter.makeFilter(),
    },
  });

export const queryFindPerformers = (filter: ListFilterModel) =>
  client.query<GQL.FindPerformersQuery>({
    query: GQL.FindPerformersDocument,
    variables: {
      filter: filter.makeFindFilter(),
      performer_filter: filter.makeFilter(),
    },
  });

export const useFindGallery = (id: string) => {
  const skip = id === "new";
  return GQL.useFindGalleryQuery({ variables: { id }, skip });
};
export const useFindScene = (id: string) =>
  GQL.useFindSceneQuery({ variables: { id } });
export const useSceneStreams = (id: string) =>
  GQL.useSceneStreamsQuery({ variables: { id } });

export const useFindImage = (id: string) =>
  GQL.useFindImageQuery({ variables: { id } });

export const useFindPerformer = (id: string) => {
  const skip = id === "new";
  return GQL.useFindPerformerQuery({ variables: { id }, skip });
};
export const useFindStudio = (id: string) => {
  const skip = id === "new";
  return GQL.useFindStudioQuery({ variables: { id }, skip });
};
export const useFindMovie = (id: string) => {
  const skip = id === "new";
  return GQL.useFindMovieQuery({ variables: { id }, skip });
};
export const useFindTag = (id: string) => {
  const skip = id === "new";
  return GQL.useFindTagQuery({ variables: { id }, skip });
};

const sceneMarkerMutationImpactedQueries = [
  GQL.FindSceneDocument,
  GQL.FindScenesDocument,
  GQL.FindSceneMarkersDocument,
  GQL.MarkerStringsDocument,
];

export const useSceneMarkerCreate = () =>
  GQL.useSceneMarkerCreateMutation({
    refetchQueries: getQueryNames([GQL.FindSceneDocument]),
    update: deleteCache(sceneMarkerMutationImpactedQueries),
  });
export const useSceneMarkerUpdate = () =>
  GQL.useSceneMarkerUpdateMutation({
    refetchQueries: getQueryNames([GQL.FindSceneDocument]),
    update: deleteCache(sceneMarkerMutationImpactedQueries),
  });
export const useSceneMarkerDestroy = () =>
  GQL.useSceneMarkerDestroyMutation({
    refetchQueries: getQueryNames([GQL.FindSceneDocument]),
    update: deleteCache(sceneMarkerMutationImpactedQueries),
  });

export const useListPerformerScrapers = () =>
  GQL.useListPerformerScrapersQuery();
export const useScrapePerformerList = (scraperId: string, q: string) =>
  GQL.useScrapePerformerListQuery({
    variables: { scraper_id: scraperId, query: q },
    skip: q === "",
  });
export const useScrapePerformer = (
  scraperId: string,
  scrapedPerformer: GQL.ScrapedPerformerInput
) =>
  GQL.useScrapePerformerQuery({
    variables: { scraper_id: scraperId, scraped_performer: scrapedPerformer },
  });

export const useListSceneScrapers = () => GQL.useListSceneScrapersQuery();

export const useListGalleryScrapers = () => GQL.useListGalleryScrapersQuery();

export const useListMovieScrapers = () => GQL.useListMovieScrapersQuery();

export const useScrapeFreeonesPerformers = (q: string) =>
  GQL.useScrapeFreeonesPerformersQuery({ variables: { q } });

export const usePlugins = () => GQL.usePluginsQuery();
export const usePluginTasks = () => GQL.usePluginTasksQuery();

export const useMarkerStrings = () => GQL.useMarkerStringsQuery();
export const useAllTags = () => GQL.useAllTagsQuery();
export const useAllTagsForFilter = () => GQL.useAllTagsForFilterQuery();
export const useAllPerformersForFilter = () =>
  GQL.useAllPerformersForFilterQuery();
export const useAllStudiosForFilter = () => GQL.useAllStudiosForFilterQuery();
export const useAllMoviesForFilter = () => GQL.useAllMoviesForFilterQuery();
export const useStats = () => GQL.useStatsQuery();
export const useVersion = () => GQL.useVersionQuery();
export const useLatestVersion = () =>
  GQL.useLatestVersionQuery({
    notifyOnNetworkStatusChange: true,
    errorPolicy: "ignore",
  });

export const useConfiguration = () => GQL.useConfigurationQuery();
export const mutateSetup = (input: GQL.SetupInput) =>
  client.mutate<GQL.SetupMutation>({
    mutation: GQL.SetupDocument,
    variables: { input },
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const mutateMigrate = (input: GQL.MigrateInput) =>
  client.mutate<GQL.MigrateMutation>({
    mutation: GQL.MigrateDocument,
    variables: { input },
  });

export const useDirectory = (path?: string) =>
  GQL.useDirectoryQuery({ variables: { path } });

const performerMutationImpactedQueries = [
  GQL.FindPerformersDocument,
  GQL.FindSceneDocument,
  GQL.FindScenesDocument,
  GQL.AllPerformersForFilterDocument,
];

export const usePerformerCreate = () =>
  GQL.usePerformerCreateMutation({
    refetchQueries: getQueryNames([
      GQL.FindPerformersDocument,
      GQL.AllPerformersForFilterDocument,
    ]),
    update: deleteCache([
      GQL.FindPerformersDocument,
      GQL.AllPerformersForFilterDocument,
    ]),
  });
export const usePerformerUpdate = () =>
  GQL.usePerformerUpdateMutation({
    update: deleteCache(performerMutationImpactedQueries),
  });

export const useBulkPerformerUpdate = (input: GQL.BulkPerformerUpdateInput) =>
  GQL.useBulkPerformerUpdateMutation({
    variables: {
      input,
    },
    update: deleteCache(performerMutationImpactedQueries),
  });

export const usePerformerDestroy = () =>
  GQL.usePerformerDestroyMutation({
    refetchQueries: getQueryNames([
      GQL.FindPerformersDocument,
      GQL.AllPerformersForFilterDocument,
    ]),
    update: deleteCache(performerMutationImpactedQueries),
  });

export const usePerformersDestroy = (
  variables: GQL.PerformersDestroyMutationVariables
) =>
  GQL.usePerformersDestroyMutation({
    variables,
    refetchQueries: getQueryNames([
      GQL.FindPerformersDocument,
      GQL.AllPerformersForFilterDocument,
    ]),
    update: deleteCache(performerMutationImpactedQueries),
  });

const sceneMutationImpactedQueries = [
  GQL.FindPerformerDocument,
  GQL.FindPerformersDocument,
  GQL.FindScenesDocument,
  GQL.FindSceneMarkersDocument,
  GQL.FindStudioDocument,
  GQL.FindStudiosDocument,
  GQL.FindMovieDocument,
  GQL.FindMoviesDocument,
  GQL.FindTagDocument,
  GQL.FindTagsDocument,
  GQL.AllTagsDocument,
];

export const useSceneUpdate = () =>
  GQL.useSceneUpdateMutation({
    update: deleteCache(sceneMutationImpactedQueries),
  });

export const useBulkSceneUpdate = (input: GQL.BulkSceneUpdateInput) =>
  GQL.useBulkSceneUpdateMutation({
    variables: {
      input,
    },
    update: deleteCache(sceneMutationImpactedQueries),
  });

export const useScenesUpdate = (input: GQL.SceneUpdateInput[]) =>
  GQL.useScenesUpdateMutation({ variables: { input } });

type SceneOMutation =
  | GQL.SceneIncrementOMutation
  | GQL.SceneDecrementOMutation
  | GQL.SceneResetOMutation;
const updateSceneO = (
  id: string,
  cache: ApolloCache<SceneOMutation>,
  updatedOCount?: number
) => {
  const scene = cache.readQuery<
    GQL.FindSceneQuery,
    GQL.FindSceneQueryVariables
  >({
    query: GQL.FindSceneDocument,
    variables: { id },
  });
  if (updatedOCount === undefined || !scene?.findScene) return;

  cache.writeQuery<GQL.FindSceneQuery, GQL.FindSceneQueryVariables>({
    query: GQL.FindSceneDocument,
    variables: { id },
    data: {
      ...scene,
      findScene: {
        ...scene.findScene,
        o_counter: updatedOCount,
      },
    },
  });
};

export const useSceneIncrementO = (id: string) =>
  GQL.useSceneIncrementOMutation({
    variables: { id },
    update: (cache, data) =>
      updateSceneO(id, cache, data.data?.sceneIncrementO),
  });

export const useSceneDecrementO = (id: string) =>
  GQL.useSceneDecrementOMutation({
    variables: { id },
    update: (cache, data) =>
      updateSceneO(id, cache, data.data?.sceneDecrementO),
  });

export const useSceneResetO = (id: string) =>
  GQL.useSceneResetOMutation({
    variables: { id },
    update: (cache, data) => updateSceneO(id, cache, data.data?.sceneResetO),
  });

export const useSceneDestroy = (input: GQL.SceneDestroyInput) =>
  GQL.useSceneDestroyMutation({
    variables: input,
    update: deleteCache(sceneMutationImpactedQueries),
  });

export const useScenesDestroy = (input: GQL.ScenesDestroyInput) =>
  GQL.useScenesDestroyMutation({
    variables: input,
    update: deleteCache(sceneMutationImpactedQueries),
  });

export const useSceneGenerateScreenshot = () =>
  GQL.useSceneGenerateScreenshotMutation({
    update: deleteCache([GQL.FindScenesDocument]),
  });

const imageMutationImpactedQueries = [
  GQL.FindPerformerDocument,
  GQL.FindPerformersDocument,
  GQL.FindImagesDocument,
  GQL.FindStudioDocument,
  GQL.FindStudiosDocument,
  GQL.FindTagDocument,
  GQL.FindTagsDocument,
  GQL.AllTagsDocument,
  GQL.FindGalleryDocument,
  GQL.FindGalleriesDocument,
];

export const useImageUpdate = () =>
  GQL.useImageUpdateMutation({
    update: deleteCache(imageMutationImpactedQueries),
  });

export const useBulkImageUpdate = () =>
  GQL.useBulkImageUpdateMutation({
    update: deleteCache(imageMutationImpactedQueries),
  });

export const useImagesDestroy = (input: GQL.ImagesDestroyInput) =>
  GQL.useImagesDestroyMutation({
    variables: input,
    update: deleteCache(imageMutationImpactedQueries),
  });

type ImageOMutation =
  | GQL.ImageIncrementOMutation
  | GQL.ImageDecrementOMutation
  | GQL.ImageResetOMutation;
const updateImageO = (
  id: string,
  cache: ApolloCache<ImageOMutation>,
  updatedOCount?: number
) => {
  const image = cache.readQuery<
    GQL.FindImageQuery,
    GQL.FindImageQueryVariables
  >({
    query: GQL.FindImageDocument,
    variables: { id },
  });
  if (updatedOCount === undefined || !image?.findImage) return;

  cache.writeQuery<GQL.FindImageQuery, GQL.FindImageQueryVariables>({
    query: GQL.FindImageDocument,
    variables: { id },
    data: {
      findImage: {
        ...image.findImage,
        o_counter: updatedOCount,
      },
    },
  });
};

export const useImageIncrementO = (id: string) =>
  GQL.useImageIncrementOMutation({
    variables: { id },
    update: (cache, data) =>
      updateImageO(id, cache, data.data?.imageIncrementO),
  });

export const useImageDecrementO = (id: string) =>
  GQL.useImageDecrementOMutation({
    variables: { id },
    update: (cache, data) =>
      updateImageO(id, cache, data.data?.imageDecrementO),
  });

export const useImageResetO = (id: string) =>
  GQL.useImageResetOMutation({
    variables: { id },
    update: (cache, data) => updateImageO(id, cache, data.data?.imageResetO),
  });

const galleryMutationImpactedQueries = [
  GQL.FindPerformerDocument,
  GQL.FindPerformersDocument,
  GQL.FindImagesDocument,
  GQL.FindStudioDocument,
  GQL.FindStudiosDocument,
  GQL.FindTagDocument,
  GQL.FindTagsDocument,
  GQL.AllTagsDocument,
  GQL.FindGalleryDocument,
  GQL.FindGalleriesDocument,
];

export const useGalleryCreate = () =>
  GQL.useGalleryCreateMutation({
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const useGalleryUpdate = () =>
  GQL.useGalleryUpdateMutation({
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const useBulkGalleryUpdate = () =>
  GQL.useBulkGalleryUpdateMutation({
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const useGalleryDestroy = (input: GQL.GalleryDestroyInput) =>
  GQL.useGalleryDestroyMutation({
    variables: input,
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const mutateAddGalleryImages = (input: GQL.GalleryAddInput) =>
  client.mutate<GQL.AddGalleryImagesMutation>({
    mutation: GQL.AddGalleryImagesDocument,
    variables: input,
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const mutateRemoveGalleryImages = (input: GQL.GalleryRemoveInput) =>
  client.mutate<GQL.RemoveGalleryImagesMutation>({
    mutation: GQL.RemoveGalleryImagesDocument,
    variables: input,
    update: deleteCache(galleryMutationImpactedQueries),
  });

export const studioMutationImpactedQueries = [
  GQL.FindStudiosDocument,
  GQL.FindSceneDocument,
  GQL.FindScenesDocument,
  GQL.AllStudiosForFilterDocument,
];

export const useStudioCreate = (input: GQL.StudioCreateInput) =>
  GQL.useStudioCreateMutation({
    variables: { input },
    refetchQueries: getQueryNames([GQL.AllStudiosForFilterDocument]),
    update: deleteCache([
      GQL.FindStudiosDocument,
      GQL.AllStudiosForFilterDocument,
    ]),
  });

export const useStudioUpdate = () =>
  GQL.useStudioUpdateMutation({
    update: deleteCache(studioMutationImpactedQueries),
  });

export const useStudioDestroy = (input: GQL.StudioDestroyInput) =>
  GQL.useStudioDestroyMutation({
    variables: input,
    update: deleteCache(studioMutationImpactedQueries),
  });

export const useStudiosDestroy = (input: GQL.StudiosDestroyMutationVariables) =>
  GQL.useStudiosDestroyMutation({
    variables: input,
    update: deleteCache(studioMutationImpactedQueries),
  });

export const movieMutationImpactedQueries = [
  GQL.FindSceneDocument,
  GQL.FindScenesDocument,
  GQL.FindMoviesDocument,
  GQL.AllMoviesForFilterDocument,
];

export const useMovieCreate = () =>
  GQL.useMovieCreateMutation({
    update: deleteCache([
      GQL.FindMoviesDocument,
      GQL.AllMoviesForFilterDocument,
    ]),
  });

export const useMovieUpdate = () =>
  GQL.useMovieUpdateMutation({
    update: deleteCache(movieMutationImpactedQueries),
  });

export const useMovieDestroy = (input: GQL.MovieDestroyInput) =>
  GQL.useMovieDestroyMutation({
    variables: input,
    update: deleteCache(movieMutationImpactedQueries),
  });

export const useMoviesDestroy = (input: GQL.MoviesDestroyMutationVariables) =>
  GQL.useMoviesDestroyMutation({
    variables: input,
    update: deleteCache(movieMutationImpactedQueries),
  });

export const tagMutationImpactedQueries = [
  GQL.FindSceneDocument,
  GQL.FindScenesDocument,
  GQL.FindSceneMarkersDocument,
  GQL.AllTagsDocument,
  GQL.AllTagsForFilterDocument,
  GQL.FindTagsDocument,
];

export const useTagCreate = () =>
  GQL.useTagCreateMutation({
    refetchQueries: getQueryNames([
      GQL.AllTagsDocument,
      GQL.AllTagsForFilterDocument,
      GQL.FindTagsDocument,
    ]),
    update: deleteCache([
      GQL.FindTagsDocument,
      GQL.AllTagsDocument,
      GQL.AllTagsForFilterDocument,
    ]),
  });
export const useTagUpdate = () =>
  GQL.useTagUpdateMutation({
    update: deleteCache(tagMutationImpactedQueries),
  });
export const useTagDestroy = (input: GQL.TagDestroyInput) =>
  GQL.useTagDestroyMutation({
    variables: input,
    update: deleteCache(tagMutationImpactedQueries),
  });

export const useTagsDestroy = (input: GQL.TagsDestroyMutationVariables) =>
  GQL.useTagsDestroyMutation({
    variables: input,
    update: deleteCache(tagMutationImpactedQueries),
  });

export const savedFilterMutationImpactedQueries = [
  GQL.FindSavedFiltersDocument,
];

export const useSaveFilter = () =>
  GQL.useSaveFilterMutation({
    update: deleteCache(savedFilterMutationImpactedQueries),
  });

export const savedFilterDefaultMutationImpactedQueries = [
  GQL.FindDefaultFilterDocument,
];

export const useSetDefaultFilter = () =>
  GQL.useSetDefaultFilterMutation({
    update: deleteCache(savedFilterDefaultMutationImpactedQueries),
  });

export const useSavedFilterDestroy = () =>
  GQL.useDestroySavedFilterMutation({
    update: deleteCache(savedFilterMutationImpactedQueries),
  });

export const useTagsMerge = () =>
  GQL.useTagsMergeMutation({
    update: deleteCache(tagMutationImpactedQueries),
  });

export const useConfigureGeneral = (input: GQL.ConfigGeneralInput) =>
  GQL.useConfigureGeneralMutation({
    variables: { input },
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const useConfigureInterface = (input: GQL.ConfigInterfaceInput) =>
  GQL.useConfigureInterfaceMutation({
    variables: { input },
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const useGenerateAPIKey = () =>
  GQL.useGenerateApiKeyMutation({
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const useJobsSubscribe = () => GQL.useJobsSubscribeSubscription();

export const useConfigureDLNA = () =>
  GQL.useConfigureDlnaMutation({
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const useEnableDLNA = () => GQL.useEnableDlnaMutation();

export const useDisableDLNA = () => GQL.useDisableDlnaMutation();

export const useAddTempDLNAIP = () => GQL.useAddTempDlnaipMutation();

export const useRemoveTempDLNAIP = () => GQL.useRemoveTempDlnaipMutation();

export const useLoggingSubscribe = () => GQL.useLoggingSubscribeSubscription();

export const useConfigureScraping = (input: GQL.ConfigScrapingInput) =>
  GQL.useConfigureScrapingMutation({
    variables: { input },
    refetchQueries: getQueryNames([GQL.ConfigurationDocument]),
    update: deleteCache([GQL.ConfigurationDocument]),
  });

export const querySystemStatus = () =>
  client.query<GQL.SystemStatusQuery>({
    query: GQL.SystemStatusDocument,
    fetchPolicy: "no-cache",
  });

export const useSystemStatus = () =>
  GQL.useSystemStatusQuery({
    fetchPolicy: "no-cache",
  });

export const useLogs = () =>
  GQL.useLogsQuery({
    fetchPolicy: "no-cache",
  });

export const useJobQueue = () =>
  GQL.useJobQueueQuery({
    fetchPolicy: "no-cache",
  });

export const mutateStopJob = (jobID: string) =>
  client.mutate<GQL.StopJobMutation>({
    mutation: GQL.StopJobDocument,
    variables: {
      job_id: jobID,
    },
  });

export const useDLNAStatus = () =>
  GQL.useDlnaStatusQuery({
    fetchPolicy: "no-cache",
  });

export const queryScrapeFreeones = (performerName: string) =>
  client.query<GQL.ScrapeFreeonesQuery>({
    query: GQL.ScrapeFreeonesDocument,
    variables: {
      performer_name: performerName,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapePerformer = (
  scraperId: string,
  scrapedPerformer: GQL.ScrapedPerformerInput
) =>
  client.query<GQL.ScrapePerformerQuery>({
    query: GQL.ScrapePerformerDocument,
    variables: {
      scraper_id: scraperId,
      scraped_performer: scrapedPerformer,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapePerformerURL = (url: string) =>
  client.query<GQL.ScrapePerformerUrlQuery>({
    query: GQL.ScrapePerformerUrlDocument,
    variables: {
      url,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapeSceneURL = (url: string) =>
  client.query<GQL.ScrapeSceneUrlQuery>({
    query: GQL.ScrapeSceneUrlDocument,
    variables: {
      url,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapeGalleryURL = (url: string) =>
  client.query<GQL.ScrapeGalleryUrlQuery>({
    query: GQL.ScrapeGalleryUrlDocument,
    variables: {
      url,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapeMovieURL = (url: string) =>
  client.query<GQL.ScrapeMovieUrlQuery>({
    query: GQL.ScrapeMovieUrlDocument,
    variables: {
      url,
    },
    fetchPolicy: "network-only",
  });

export const queryScrapeScene = (
  scraperId: string,
  scene: GQL.SceneUpdateInput
) =>
  client.query<GQL.ScrapeSceneQuery>({
    query: GQL.ScrapeSceneDocument,
    variables: {
      scraper_id: scraperId,
      scene,
    },
    fetchPolicy: "network-only",
  });

export const queryStashBoxScene = (stashBoxIndex: number, sceneID: string) =>
  client.query<GQL.QueryStashBoxSceneQuery>({
    query: GQL.QueryStashBoxSceneDocument,
    variables: {
      input: {
        stash_box_index: stashBoxIndex,
        scene_ids: [sceneID],
      },
    },
  });

export const queryStashBoxPerformer = (
  stashBoxIndex: number,
  performerID: string
) =>
  client.query<GQL.QueryStashBoxPerformerQuery>({
    query: GQL.QueryStashBoxPerformerDocument,
    variables: {
      input: {
        stash_box_index: stashBoxIndex,
        performer_ids: [performerID],
      },
    },
  });

export const queryScrapeGallery = (
  scraperId: string,
  gallery: GQL.GalleryUpdateInput
) =>
  client.query<GQL.ScrapeGalleryQuery>({
    query: GQL.ScrapeGalleryDocument,
    variables: {
      scraper_id: scraperId,
      gallery,
    },
    fetchPolicy: "network-only",
  });

export const mutateReloadScrapers = () =>
  client.mutate<GQL.ReloadScrapersMutation>({
    mutation: GQL.ReloadScrapersDocument,
    refetchQueries: [
      GQL.refetchListMovieScrapersQuery(),
      GQL.refetchListPerformerScrapersQuery(),
      GQL.refetchListSceneScrapersQuery(),
    ],
  });

export const mutateReloadPlugins = () =>
  client.mutate<GQL.ReloadPluginsMutation>({
    mutation: GQL.ReloadPluginsDocument,
    refetchQueries: [GQL.refetchPluginsQuery(), GQL.refetchPluginTasksQuery()],
  });

export const mutateRunPluginTask = (
  pluginId: string,
  taskName: string,
  args?: GQL.PluginArgInput[]
) =>
  client.mutate<GQL.RunPluginTaskMutation>({
    mutation: GQL.RunPluginTaskDocument,
    variables: { plugin_id: pluginId, task_name: taskName, args },
  });

export const mutateMetadataScan = (input: GQL.ScanMetadataInput) =>
  client.mutate<GQL.MetadataScanMutation>({
    mutation: GQL.MetadataScanDocument,
    variables: { input },
  });

export const mutateMetadataAutoTag = (input: GQL.AutoTagMetadataInput) =>
  client.mutate<GQL.MetadataAutoTagMutation>({
    mutation: GQL.MetadataAutoTagDocument,
    variables: { input },
  });

export const mutateMetadataGenerate = (input: GQL.GenerateMetadataInput) =>
  client.mutate<GQL.MetadataGenerateMutation>({
    mutation: GQL.MetadataGenerateDocument,
    variables: { input },
  });

export const mutateMetadataClean = (input: GQL.CleanMetadataInput) =>
  client.mutate<GQL.MetadataCleanMutation>({
    mutation: GQL.MetadataCleanDocument,
    variables: { input },
  });

export const mutateMigrateHashNaming = () =>
  client.mutate<GQL.MigrateHashNamingMutation>({
    mutation: GQL.MigrateHashNamingDocument,
  });

export const mutateMetadataExport = () =>
  client.mutate<GQL.MetadataExportMutation>({
    mutation: GQL.MetadataExportDocument,
  });

export const mutateExportObjects = (input: GQL.ExportObjectsInput) =>
  client.mutate<GQL.ExportObjectsMutation>({
    mutation: GQL.ExportObjectsDocument,
    variables: { input },
  });

export const mutateMetadataImport = () =>
  client.mutate<GQL.MetadataImportMutation>({
    mutation: GQL.MetadataImportDocument,
  });

export const mutateImportObjects = (input: GQL.ImportObjectsInput) =>
  client.mutate<GQL.ImportObjectsMutation>({
    mutation: GQL.ImportObjectsDocument,
    variables: { input },
  });

export const mutateBackupDatabase = (input: GQL.BackupDatabaseInput) =>
  client.mutate<GQL.BackupDatabaseMutation>({
    mutation: GQL.BackupDatabaseDocument,
    variables: { input },
  });

export const mutateStashBoxBatchPerformerTag = (
  input: GQL.StashBoxBatchPerformerTagInput
) =>
  client.mutate<GQL.StashBoxBatchPerformerTagMutation>({
    mutation: GQL.StashBoxBatchPerformerTagDocument,
    variables: { input },
  });

export const querySceneByPathRegex = (filter: GQL.FindFilterType) =>
  client.query<GQL.FindScenesByPathRegexQuery>({
    query: GQL.FindScenesByPathRegexDocument,
    variables: { filter },
  });

export const queryParseSceneFilenames = (
  filter: GQL.FindFilterType,
  config: GQL.SceneParserInput
) =>
  client.query<GQL.ParseSceneFilenamesQuery>({
    query: GQL.ParseSceneFilenamesDocument,
    variables: { filter, config },
    fetchPolicy: "network-only",
  });

export const makePerformerCreateInput = (
  toCreate: GQL.ScrapedScenePerformer
) => {
  const input: GQL.PerformerCreateInput = {
    name: toCreate.name,
    url: toCreate.url,
    gender: stringToGender(toCreate.gender),
    birthdate: toCreate.birthdate,
    ethnicity: toCreate.ethnicity,
    country: toCreate.country,
    eye_color: toCreate.eye_color,
    height: toCreate.height,
    measurements: toCreate.measurements,
    fake_tits: toCreate.fake_tits,
    career_length: toCreate.career_length,
    tattoos: toCreate.tattoos,
    piercings: toCreate.piercings,
    aliases: toCreate.aliases,
    twitter: toCreate.twitter,
    instagram: toCreate.instagram,
    tag_ids: filterData((toCreate.tags ?? []).map((t) => t.stored_id)),
    image:
      (toCreate.images ?? []).length > 0
        ? (toCreate.images ?? [])[0]
        : undefined,
    details: toCreate.details,
    death_date: toCreate.death_date,
    hair_color: toCreate.hair_color,
    weight: toCreate.weight ? Number(toCreate.weight) : undefined,
  };
  return input;
};

export const stashBoxSceneQuery = (searchVal: string, stashBoxIndex: number) =>
  client?.query<
    GQL.QueryStashBoxSceneQuery,
    GQL.QueryStashBoxSceneQueryVariables
  >({
    query: GQL.QueryStashBoxSceneDocument,
    variables: { input: { q: searchVal, stash_box_index: stashBoxIndex } },
  });

export const stashBoxPerformerQuery = (
  searchVal: string,
  stashBoxIndex: number
) =>
  client?.query<
    GQL.QueryStashBoxPerformerQuery,
    GQL.QueryStashBoxPerformerQueryVariables
  >({
    query: GQL.QueryStashBoxPerformerDocument,
    variables: { input: { q: searchVal, stash_box_index: stashBoxIndex } },
  });

export const stashBoxSceneBatchQuery = (
  sceneIds: string[],
  stashBoxIndex: number
) =>
  client?.query<
    GQL.QueryStashBoxSceneQuery,
    GQL.QueryStashBoxSceneQueryVariables
  >({
    query: GQL.QueryStashBoxSceneDocument,
    variables: {
      input: { scene_ids: sceneIds, stash_box_index: stashBoxIndex },
    },
  });

export const stashBoxPerformerBatchQuery = (
  performerIds: string[],
  stashBoxIndex: number
) =>
  client?.query<
    GQL.QueryStashBoxPerformerQuery,
    GQL.QueryStashBoxPerformerQueryVariables
  >({
    query: GQL.QueryStashBoxPerformerDocument,
    variables: {
      input: { performer_ids: performerIds, stash_box_index: stashBoxIndex },
    },
  });
