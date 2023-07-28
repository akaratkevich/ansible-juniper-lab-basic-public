cleanup:
    @sudo clab destroy -t ./clab/topology.yml --cleanup

run:
    @sudo clab deploy -t ./clab/topology.yml
