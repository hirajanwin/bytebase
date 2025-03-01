<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <div class="flex flex-row space-x-4">
        <button
          v-if="project"
          class="px-4 cursor-pointer rounded-md text-control text-sm bg-link-hover focus:outline-none hover:underline"
          @click.prevent="goProject"
        >
          {{ project.key }}
        </button>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <MemberSelect
          v-if="scopeByPrincipal"
          class="w-72"
          :show-all="true"
          :show-system-bot="true"
          :selected-id="state.selectedPrincipalId"
          @select-principal-id="selectPrincipal"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('issue.search-issue-name')"
          @change-text="(text) => changeSearchText(text)"
        />
      </div>
    </div>
    <IssueTable
      :left-bordered="false"
      :right-bordered="false"
      :top-bordered="true"
      :bottom-bordered="true"
      :issue-section-list="sectionList"
    />
  </div>
</template>

<script lang="ts">
import { useRoute, useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import { IssueTable } from "../components/Issue";
import MemberSelect from "../components/MemberSelect.vue";
import { EMPTY_ID, Environment, Issue, PrincipalId, ProjectId } from "../types";
import {
  reactive,
  ref,
  computed,
  onMounted,
  watchEffect,
  defineComponent,
  watch,
} from "vue";
import { activeEnvironment, isDBAOrOwner, projectSlug } from "../utils";
import { BBTableSectionDataSource } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import {
  useCurrentUser,
  useEnvironmentStore,
  useIssueStore,
  useProjectStore,
} from "@/store";

interface LocalState {
  showOpen: boolean;
  showClosed: boolean;
  openList: Issue[];
  closedList: Issue[];
  searchText: string;
  selectedPrincipalId: PrincipalId;
  selectedEnvironment?: Environment;
  selectedProjectId?: ProjectId;
}

export default defineComponent({
  name: "IssueDashboard",
  components: { EnvironmentTabFilter, IssueTable, MemberSelect },
  setup() {
    const { t } = useI18n();
    const searchField = ref();

    const issueStore = useIssueStore();
    const router = useRouter();
    const route = useRoute();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    const statusList: string[] = route.query.status
      ? (route.query.status as string).split(",")
      : [];

    // Applies principal scope if we explicitly specify user in the query parameter
    // or project is NOT present in the query parameter.
    // In other words, if only project is present in the query parameter, then
    // we do NOT apply principal scope, which is the case if we want to list all issues
    // for a particular project.
    // Note: We do not use computed, otherwise it will cause prepareIssueList to refetch everytime we click environment tab
    const scopeByPrincipal = !route.query.user || !route.query.project;

    const state = reactive<LocalState>({
      showOpen: statusList.length == 0 || statusList.includes("open"),
      showClosed: statusList.length == 0 || statusList.includes("closed"),
      openList: [],
      closedList: [],
      searchText: "",
      selectedPrincipalId: currentUser.value.id, // initialized below
      selectedEnvironment: route.query.environment
        ? useEnvironmentStore().getEnvironmentById(
            parseInt(route.query.environment as string, 10)
          )
        : undefined,
      selectedProjectId: route.query.project
        ? parseInt(route.query.project as string)
        : undefined,
    });
    // initialize the selectedPrincipalId
    const initializeSelectedPrincipalIdFromQuery = (): PrincipalId => {
      const id = parseInt(route.query.user as string, 10);
      if (id >= 0) {
        return id;
      }
      return isDBAOrOwner(currentUser.value.role)
        ? EMPTY_ID // default to 'All' if current user is owner or DBA
        : currentUser.value.id; // default to current user otherwise
    };

    watch(
      () => route.query.user,
      () => {
        state.selectedPrincipalId = initializeSelectedPrincipalIdFromQuery();
      },
      { immediate: true }
    );

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const project = computed(() => {
      if (state.selectedProjectId) {
        return projectStore.getProjectById(state.selectedProjectId);
      }
      return undefined;
    });

    const prepareIssueList = () => {
      // We call open and close separately because normally the number of open issues is limited
      // while the closed issues could be a lot.
      // When "All" is selected, the userId is eventually ignored by `fetchIssueList` because
      // it's 0 (EMPTY_ID), so we need not to check the special value here.
      if (state.showOpen) {
        issueStore
          .fetchIssueList({
            issueStatusList: ["OPEN"],
            userId: scopeByPrincipal ? state.selectedPrincipalId : undefined,
            projectId: state.selectedProjectId,
          })
          .then((issueList: Issue[]) => {
            state.openList = issueList;
          });
      }

      if (state.showClosed) {
        issueStore
          .fetchIssueList({
            issueStatusList: ["DONE", "CANCELED"],
            userId: scopeByPrincipal ? state.selectedPrincipalId : undefined,
            projectId: state.selectedProjectId,
          })
          .then((issueList: Issue[]) => {
            state.closedList = issueList;
          });
      }
    };

    watchEffect(prepareIssueList);

    const filteredList = (list: Issue[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((issue) => {
        return (
          (!state.selectedEnvironment ||
            activeEnvironment(issue.pipeline).id ===
              state.selectedEnvironment.id) &&
          (!state.searchText ||
            issue.name.toLowerCase().includes(state.searchText.toLowerCase()))
        );
      });
    };

    const sectionList = computed((): BBTableSectionDataSource<Issue>[] => {
      const list = [];
      if (state.showOpen) {
        list.push({
          title: t("issue.table.open"),
          list: filteredList(state.openList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        });
      }
      if (state.showClosed) {
        list.push({
          title: t("issue.table.closed"),
          list: filteredList(state.closedList).sort((a, b) => {
            return b.updatedTs - a.updatedTs;
          }),
        });
      }
      return list;
    });

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.issue",
          query: {
            ...route.query,
            environment: environment.id,
          },
        });
      } else {
        router.replace({
          name: "workspace.issue",
          query: {
            ...route.query,
            environment: undefined,
          },
        });
      }
    };

    const selectPrincipal = (principalId: PrincipalId) => {
      state.selectedPrincipalId = principalId;
      router.replace({
        name: "workspace.issue",
        query: {
          ...route.query,
          user: principalId,
        },
      });
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const goProject = () => {
      if (!project.value) return;
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: projectSlug(project.value),
        },
      });
    };

    return {
      searchField,
      state,
      scopeByPrincipal,
      project,
      sectionList,
      selectEnvironment,
      selectPrincipal,
      changeSearchText,
      goProject,
    };
  },
});
</script>
