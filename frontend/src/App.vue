<template>
  <div class="flex flex-col h-screen overflow-hidden relative">
    
    <!-- Top Header -->
    <Header />

    <!-- Toast Notifications -->
    <Toast />

    <!-- Main Workspace Body -->
    <main class="flex-1 flex overflow-hidden">
      <!-- Left Sidebar -->
      <Sidebar />

      <!-- Right Column: Switchable Views -->
      <section class="flex-1 flex flex-col overflow-hidden bg-slate-900">
        <ProgressBar />
        <PagesTab />
        <AnalyticsTab />
        <ImagesTab />
        <FontsTab />
        <IframesTab />
        <FormsTab />
      </section>
    </main>

    <!-- Modals & Drawers -->
    <LogDrawer />
    <SettingsModal />
    <ImageComparison />
    <ImageStudio />
    <PageDetail />
    <WpExport />
    <WpApply />
    <ProfileModal />
    <FormDetail />
    <PokemonModal />

  </div>
</template>

<script>
import { onMounted } from 'vue';
import { useApp } from '@/store/app';

import Header from './components/layout/Header.vue';
import Toast from './components/layout/Toast.vue';
import Sidebar from './components/layout/Sidebar.vue';
import ProgressBar from './components/layout/ProgressBar.vue';

import PagesTab from './components/tabs/PagesTab.vue';
import AnalyticsTab from './components/tabs/AnalyticsTab.vue';
import ImagesTab from './components/tabs/ImagesTab.vue';
import FontsTab from './components/tabs/FontsTab.vue';
import IframesTab from './components/tabs/IframesTab.vue';
import FormsTab from './components/tabs/FormsTab.vue';

import LogDrawer from './components/modals/LogDrawer.vue';
import SettingsModal from './components/modals/SettingsModal.vue';
import ImageComparison from './components/modals/ImageComparison.vue';
import ImageStudio from './components/modals/ImageStudio.vue';
import PageDetail from './components/modals/PageDetail.vue';
import WpExport from './components/modals/WpExport.vue';
import WpApply from './components/modals/WpApply.vue';
import ProfileModal from './components/modals/ProfileModal.vue';
import FormDetail from './components/modals/FormDetail.vue';
import PokemonModal from './components/modals/PokemonModal.vue';

export default {
  name: 'App',
  components: {
    Header,
    Toast,
    Sidebar,
    ProgressBar,
    PagesTab,
    AnalyticsTab,
    ImagesTab,
    FontsTab,
    IframesTab,
    FormsTab,
    LogDrawer,
    SettingsModal,
    ImageComparison,
    ImageStudio,
    PageDetail,
    WpExport,
    WpApply,
    ProfileModal,
    FormDetail,
    PokemonModal,
  },
  setup() {
    const app = useApp();

    onMounted(() => {
      app.initApp();
    });

    return app;
  }
};
</script>
