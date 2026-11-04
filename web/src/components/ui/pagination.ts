/** Options accepted by `CfPagination`, mirroring Firefish's `Paging`. */
export interface Paging {
	endpoint: string;
	params?: Record<string, unknown>;
	limit?: number;
	noPaging?: boolean;
	reversed?: boolean;
}
