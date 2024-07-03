<script lang="ts">
  export let options: any[] = [];
  export let setOption = (item: any) => {
    console.log(item);
  };
</script>

<div class="dropdown">
  <button
    class="btn btn-primary dropdown-toggle"
    type="button"
    id="dropdownMenuButton"
    data-toggle="dropdown"
    aria-haspopup="true"
    aria-expanded="false"
  >
    Dropdown button
  </button>
  <div class="dropdown-menu" aria-labelledby="dropdownMenuButton">
    {#each options as option}
      <a class="dropdown-item" href={option.value}>{option.Label}</a>
    {/each}
  </div>
</div>

<style>
  .dropdown {
    position: relative;
    display: inline-block;
  }

  .dropdown-menu {
    display: none;
    position: absolute;
    z-index: 1;
  }

  .dropdown-menu.show {
    display: block;
  }
</style>
