<template>
  <div class="flex gap-x-20">
    <SQLReviewSidebar :selected-rule-list="selectedRuleList" />
    <div class="flex-1">
      <SQLReviewTemplates
        v-if="selectedRuleList.length === 0"
        :required="true"
        :template-list="templateList"
        :title="$t('sql-review.create.basic-info.choose-template')"
        @select="(index) => onTemplateApply(index)"
      />
      <div v-if="selectedRuleList.length > 0" class="mb-5">
        <div
          class="flex cursor-pointer items-center px-2 text-indigo-500"
          @click="state.openTemplate = !state.openTemplate"
        >
          <heroicons-solid:chevron-right
            class="w-5 h-5 transform transition-all"
            :class="state.openTemplate ? 'rotate-90' : ''"
          />
          <span class="ml-1 text-sm font-medium">
            {{ $t("sql-review.create.configure-rule.change-template") }}
          </span>
        </div>
        <SQLReviewTemplates
          v-if="state.openTemplate"
          :required="false"
          :template-list="templateList"
          :selected-template-index="selectedTemplateIndex"
          class="mx-8 mt-5"
          @select="(index) => onTemplateApply(index)"
        />
      </div>
      <div
        :class="selectedRuleList.length > 0 ? 'border-b-1 border-gray-200' : ''"
      >
        <ul role="list" class="divide-y divide-gray-200">
          <li v-for="rule in selectedRuleList" :key="rule.type">
            <SQLRuleConfig
              :selected-rule="rule"
              :active="rule.type === state.activeRuleType"
              @activate="onRuleActivate"
              @level-change="(level) => onLevelChange(rule, level)"
              @payload-change="(val) => onPayloadChange(rule, val)"
            />
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import {
  RuleType,
  RuleLevel,
  RuleTemplate,
  RuleConfigComponent,
  SQLReviewPolicyTemplate,
} from "@/types/sqlReview";

interface LocalState {
  activeRuleType: RuleType | null;
  openTemplate: boolean;
}

const props = defineProps({
  selectedRuleList: {
    required: true,
    type: Object as PropType<RuleTemplate[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SQLReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
  },
});

const emit = defineEmits(["apply-template", "change"]);

const state = reactive<LocalState>({
  activeRuleType: null,
  openTemplate: false,
});

const onTemplateApply = (index: number) => {
  emit("apply-template", index);
  state.activeRuleType = null;
};

const onRuleActivate = (type: RuleType) => {
  if (type === state.activeRuleType) {
    state.activeRuleType = null;
  } else {
    state.activeRuleType = type;
  }
};

const onPayloadChange = (
  rule: RuleTemplate,
  data: (string | number | string[])[]
) => {
  if (!rule.componentList) {
    return;
  }

  const newRule: RuleTemplate = {
    ...rule,
    componentList: rule.componentList.reduce((list, component, index) => {
      let val = data[index];
      switch (component.payload.type) {
        case "STRING_ARRAY":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string[],
            },
          });
          break;
        case "NUMBER":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as number,
            },
          });
          break;
        default:
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string,
            },
          });
          break;
      }
      return list;
    }, [] as RuleConfigComponent[]),
  };

  emit("change", newRule);
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  emit("change", {
    ...rule,
    level,
  });
};
</script>
