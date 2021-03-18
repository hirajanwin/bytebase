export type BBTableColumn = {
  title: string;
};

export type BBTableSectionDataSource<T> = {
  title: string;
  list: T[];
};

export type BBStepStatus =
  | "PENDING"
  | "PENDING_ACTIVE"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED"
  | "SKIPPED";

export type BBStep = {
  title: string;
  status: BBStepStatus;
  link(): string;
};

export type BBOutlineItem = {
  name: string;
  link?: string;
  childList?: [];
};

export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";
export type BBNotificationPlacement =
  | "TOP_LEFT"
  | "TOP_RIGHT"
  | "BOTTOM_LEFT"
  | "BOTTOM_RIGHT";
